package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/bool64/ctxd"
	"github.com/bool64/zapctxd"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/image"
	"github.com/vearutop/photo-blog/internal/infra/image/sprite"
)

func main() {
	if err := Main(); err != nil {
		println(err.Error())
		os.Exit(1)
	}
}

func Main() error {
	var (
		mu        sync.Mutex
		files     []image.Data
		filesMap  = make(map[string]int) // Name to files index.
		semaphore = make(chan struct{}, runtime.NumCPU())
		log       = zapctxd.New(zapctxd.Config{ColoredOutput: true})

		listLists bool
	)

	flag.BoolVar(&listLists, "l", false, "List all available lists")
	flag.Parse()

	if listLists {
		var list []string

		err := filepath.Walk(".", func(p string, info os.FileInfo, err error) error {
			// List found, e.g. ls_1s4za3zpktl61.json.
			if strings.Contains(p, "/ls_") && strings.HasSuffix(p, ".json") {
				list = append(list, p)
			}

			return nil
		})
		if err != nil {
			return err
		}

		j, err := json.Marshal(list)
		if err != nil {
			return err
		}

		fmt.Println(string(j))

		return nil
	}

	ctx := context.Background()

	ts := thumbStorer{}

	listFn := ""
	err := filepath.Walk(".", func(p string, info os.FileInfo, err error) error {
		// List found, e.g. ls_1s4za3zpktl61.json.
		if strings.HasPrefix(p, "ls_") && strings.HasSuffix(p, ".json") {
			listFn = p
		}

		return nil
	})
	if err != nil {
		return err
	}

	if listFn == "" {
		listFn = fmt.Sprintf("ls_%s.json", uniq.Hash(rand.Int64()))
	} else {
		log.Info(ctx, "existing list found", "list", listFn)

		j, err := os.ReadFile(listFn)
		if err != nil {
			return err
		}

		err = json.Unmarshal(j, &files)
		if err != nil {
			return err
		}

		for i, f := range files {
			filesMap[f.Image.Path] = i
		}
	}

	err = filepath.Walk(".", func(p string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if strings.HasPrefix(p, "thumbs/") {
			return nil
		}

		l := strings.ToLower(p)

		if strings.HasSuffix(l, ".jpeg") || strings.HasSuffix(l, ".jpg") {
			println("processing", p)

			i, alreadyListed := filesMap[p]
			d := image.Data{}
			if alreadyListed {
				d = files[i]
			}

			semaphore <- struct{}{} // Acquire semaphore slot.
			go func() {
				defer func() {
					<-semaphore // Release semaphore slot.
				}()

				if d.Image.Path == "" {
					d.Image.Path = p
				}

				if err := d.Fill(ctx); err != nil {
					ctxd.LogError(ctx, err, log.Error)
					return
				}

				hashSuffix := fmt.Sprintf(".%s.jpg", d.Image.Hash)
				if !strings.HasSuffix(l, hashSuffix) {
					if err := os.Rename(p, p+hashSuffix); err != nil {
						ctxd.LogError(ctx, fmt.Errorf("rename with hashed suffix: %w", err), log.Error)
						return
					}

					d.Image.Path = p + hashSuffix
				}

				for i, th := range d.Thumbs {
					if th.FilePath != "" || th.SpriteFile != "" {
						continue
					}

					if err := ts.Write(&th); err != nil {
						ctxd.LogError(ctx, err, log.Error)
						continue
					}
					d.Thumbs[i] = th
				}

				mu.Lock()
				defer mu.Unlock()

				if alreadyListed {
					files[i] = d
					delete(filesMap, p)
				} else {
					files = append(files, d)
				}
			}()
		}

		println(p)
		return nil
	})
	if err != nil {
		return err
	}

	if len(filesMap) > 0 {
		result := make([]image.Data, 0, len(files)-len(filesMap))
		deleted := make([]image.Data, 0, len(filesMap))
		for _, f := range files {
			if _, ok := filesMap[f.Image.Path]; !ok {
				result = append(result, f)
			} else {
				deleted = append(deleted, f)
			}
		}

		log.Info(ctx, "deleted files in the list", "files", deleted)

		files = result
	}

	// Wait for goroutines to finish by acquiring all slots.
	for i := 0; i < cap(semaphore); i++ {
		semaphore <- struct{}{}
	}

	if err := ts.Close(); err != nil {
		return err
	}

	if j, err := json.MarshalIndent(files, "", "  "); err != nil {
		return err
	} else if err := os.WriteFile(listFn, j, 0o600); err != nil {
		return err
	}

	log.Info(ctx, "done, list written to: "+listFn)

	err = filepath.Walk(".", func(p string, info os.FileInfo, err error) error {
		if p == ".DS_Store" { // MacOS garbage.
			return os.Remove(p)
		}
		if strings.HasPrefix(p, "._") && info.Size() == 4096 { // MacOS garbage.
			return os.Remove(p)
		}
		if strings.HasPrefix(p, "thumbs/._") && info.Size() == 4096 { // MacOS garbage.
			return os.Remove(p)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

type thumbStorer struct {
	mu sync.Mutex
	z  map[photo.ThumbSize]*thumbZipWriter

	s map[int]*sprite.Vertical
}

type thumbZipWriter struct {
	zip          *zip.Writer
	idx          int
	fn           string
	bytesWritten int
}

func (t *thumbStorer) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, w := range t.z {
		if err := w.zip.Close(); err != nil {
			return err
		}
	}

	for _, sp := range t.s {
		sp.Flush()
	}

	return nil
}

func (t *thumbStorer) Write(th *photo.Thumb) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(th.Data) > 5e5 { // 500 KB
		if err := os.MkdirAll("thumbs", 0o700); err != nil {
			return err
		}
		th.FilePath = "thumbs/" + th.Hash.String() + "." + string(th.Format) + ".jpg"

		if err := os.WriteFile(th.FilePath, th.Data, 0o600); err != nil {
			return err
		}

		th.Data = nil

		return nil
	}

	if th.Width <= 600 {
		if t.s == nil {
			t.s = make(map[int]*sprite.Vertical)
		}

		w := int(th.Width)

		ts := t.s[w]
		if ts == nil {
			ts = sprite.NewVertical(w)
			ts.BaseName = "thumbs/sprite_w" + strconv.Itoa(w)
			t.s[w] = ts
		}

		spr, ofs, err := ts.AddThumb(*th)
		if err != nil {
			return err
		}

		th.SpriteFile = spr
		th.SpriteOffset = ofs
	}

	w := t.z[th.Format]

	if w == nil {
		if t.z == nil {
			t.z = map[photo.ThumbSize]*thumbZipWriter{}
		}

		w = &thumbZipWriter{}
		t.z[th.Format] = w
	}

	if w.bytesWritten+len(th.Data) > 15e6 {
		if err := w.zip.Close(); err != nil {
			return err
		}
		w.zip = nil
	}

	if w.zip == nil {
		w.idx++
		w.fn = fmt.Sprintf("thumbs/%s.%s.%d.zip", uniq.Hash(rand.Int64()), th.Format, w.idx)
		if err := os.MkdirAll("thumbs", 0o700); err != nil {
			return err
		}

		if f, err := os.Create(w.fn); err != nil {
			return err
		} else {
			w.zip = zip.NewWriter(f)
			w.bytesWritten = 0
		}
	}

	tfn := fmt.Sprintf("%s.%s.jpg", th.Hash, th.Format)
	tf, err := w.zip.CreateHeader(&zip.FileHeader{
		Name:     tfn,
		Method:   zip.Store,
		Modified: th.CreatedAt,
	})
	if err != nil {
		return err
	}

	_, err = tf.Write(th.Data)
	if err != nil {
		return err
	}

	w.bytesWritten += len(th.Data)
	th.Data = nil
	th.FilePath = path.Join(w.fn, tfn)

	return nil
}

func fileExists(f string) bool {
	_, err := os.Stat(f)
	return err == nil
}
