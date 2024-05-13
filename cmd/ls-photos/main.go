package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/internal/domain/uniq"
	"github.com/vearutop/photo-blog/internal/infra/image"
)

func main() {
	var (
		mu        sync.Mutex
		files     []image.Data
		semaphore = make(chan struct{}, runtime.NumCPU())
	)

	ctx := context.Background()

	ts := thumbStorer{}

	err := filepath.Walk(".", func(p string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if strings.HasPrefix(p, "thumbs/") {
			return nil
		}

		l := strings.ToLower(p)

		if strings.HasSuffix(l, ".jpeg") || strings.HasSuffix(l, ".jpg") {
			println("processing", p)

			d := image.Data{}

			semaphore <- struct{}{} // Acquire semaphore slot.
			go func() {
				defer func() {
					<-semaphore // Release semaphore slot.
				}()

				if err := d.Image.SetPath(ctx, p); err != nil {
					log.Println(err.Error())
					return
				}

				hashSuffix := fmt.Sprintf(".%s.jpg", d.Image.Hash)
				if !strings.HasSuffix(l, hashSuffix) {
					if err := os.Rename(p, p+hashSuffix); err != nil {
						log.Println("failed to rename with hashed suffix:", err.Error())
						return
					}

					d.Image.Path = p + hashSuffix
				}

				if err := d.Fill(ctx); err != nil {
					log.Println(err.Error())
					return
				}

				for i, th := range d.Thumbs {
					if err := ts.Write(&th); err != nil {
						log.Println(err.Error())
						continue
					}
					d.Thumbs[i] = th
				}

				mu.Lock()
				defer mu.Unlock()
				files = append(files, d)
			}()
		}

		println(p)
		return nil
	})
	if err != nil {
		log.Println(err.Error())
	}

	// Wait for goroutines to finish by acquiring all slots.
	for i := 0; i < cap(semaphore); i++ {
		semaphore <- struct{}{}
	}

	if err := ts.Close(); err != nil {
		log.Println(err.Error())
	}

	j, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		log.Println(err.Error())
	}

	listFn := fmt.Sprintf("%s.json", uniq.Hash(rand.Int64()))
	if err := os.WriteFile(listFn, j, 0o600); err != nil {
		log.Println(err.Error())
	}

	log.Println("done, list written to:", listFn)
}

type thumbStorer struct {
	mu sync.Mutex
	z  map[photo.ThumbSize]*thumbZipWriter
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

	return nil
}

func (t *thumbStorer) Write(th *photo.Thumb) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(th.Data) > 5e5 { // 500 KB
		if err := os.MkdirAll("thumbs", 0o600); err != nil {
			return err
		}
		th.FilePath = "thumbs/" + th.Hash.String() + "." + string(th.Format) + ".jpg"

		if err := os.WriteFile(th.FilePath, th.Data, 0o600); err != nil {
			return err
		}

		th.Data = nil

		return nil
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
		if err := os.MkdirAll("thumbs", 0o600); err != nil {
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
