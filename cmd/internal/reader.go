package internal

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
)

func Reader(fn string) (*bytes.Reader, error) {
	if strings.HasPrefix(fn, "http://") || strings.HasPrefix(fn, "https://") {
		resp, err := http.Get(fn)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		d, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return bytes.NewReader(d), nil
	}

	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	d, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(d), nil
}
