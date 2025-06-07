package control

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/minio/selfupdate"
	"github.com/swaggest/usecase"
	"github.com/vearutop/photo-blog/pkg/web"
)

func SelfUpdate() usecase.Interactor {
	type input struct {
		Version string `query:"version"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, in input, out *web.Page) error {
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

			err = selfupdate.Apply(tr, selfupdate.Options{})
			if err != nil {
				return err
			}
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
			println("restarting")
			// os.Exit(0)
			if err := restart(); err != nil {
				println(err)
			}
		}()

		return nil
	})

	return u
}

func restart() error {
	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %v", err)
	}

	// Prepare to restart the process
	cmd := exec.Command(execPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	// Start the new process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start new process: %v", err)
	}

	// Exit the current process
	os.Exit(0)
	return nil
}
