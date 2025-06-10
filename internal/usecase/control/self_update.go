package control

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/pkg/web"
)

func SelfUpdate() usecase.Interactor {
	type input struct {
		Version string `query:"version"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in input, out *web.Page) error {
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("get executable path: %w", err)
		}

		repo := "vearutop/photo-blog"
		url := "https://github.com/" + repo + "/releases/latest/download/" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz"

		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// Check if the response is successful
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to download: %s", resp.Status)
		}

		// Create a gzip reader
		gzr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}
		defer gzr.Close()

		// Create a tar reader
		tr := tar.NewReader(gzr)

		// Iterate through the tar archive
		for {
			header, err := tr.Next()
			if err == io.EOF {
				break // End of archive
			}
			if err != nil {
				return err
			}

			if header.Name != "photo-blog" || header.Typeflag != tar.TypeReg {
				continue
			}

			if err := performUpdate(tr, execPath); err != nil {
				return fmt.Errorf("perform update: %w", err)
			}

			break
		}

		out.ResponseWriter().Write([]byte(`
<html>
<head><title>Update complete</title></head>
<h1>Update complete</h1>

<a href="/">back to main page</a>
</html>
`))

		println("done")

		go func() {
			time.Sleep(5 * time.Second)
			println("exiting")
			os.Exit(0)
			//if err := restart(); err != nil {
			//	println(err)
			//}
		}()

		return nil
	})

	return u
}

func performUpdate(update io.Reader, execPath string) error {
	// Create a temporary file for the new binary
	out, err := os.CreateTemp("", "update-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer out.Close()

	// Copy downloaded binary to temp file
	if _, err := io.Copy(out, update); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	// Make the new binary executable
	if err := os.Chmod(out.Name(), 0o755); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}

	// Replace the current binary
	if err := os.Rename(out.Name(), execPath); err != nil {
		return fmt.Errorf("replace binary: %w", err)
	}

	return nil
}
