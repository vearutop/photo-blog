package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/evanoberholster/imagemeta"
	"github.com/vearutop/photo-blog/cmd/internal"
	"github.com/vearutop/photo-blog/internal/infra/image"
)

func main() {
	flag.Parse()

	fn := flag.Arg(0)

	r, err := internal.Reader(fn)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("github.com/evanoberholster/imagemeta")
	e, err := imagemeta.Decode(r)
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println(e.String())

	fmt.Println("github.com/dsoprea/go-exif/v3")
	m, err := image.ReadMeta(r)
	if err != nil {
		fmt.Println(err.Error())
	}

	for k, v := range m.ExifData() {
		fmt.Println(k, ":", v)
	}

	r.Seek(0, 0)
	fmt.Println("exiftool")
	cmd := exec.Command("exiftool", "-j", "-")
	cmd.Stdout = os.Stdout
	cmd.Stdin = r
	err = cmd.Run()
	if err != nil {
		fmt.Println(err.Error())
	}
}
