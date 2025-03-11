package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/vearutop/photo-blog/internal/infra/image"
	"github.com/vearutop/photo-blog/internal/infra/image/sharpness"
)

func main() {
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(flag.CommandLine.Output(), "Sharpness returns a float value that indicates how sharp the image seems.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "It uses Sobel 9x9 edge-detection to identify areas of contrast.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n")
		fmt.Fprintf(flag.CommandLine.Output(), " sharpness <file.jpg>\n")
		flag.PrintDefaults()

		return
	}

	imgSrc := flag.Arg(0)

	j, err := image.LoadJPEG(imgSrc)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(sharpness.NoiseScore(image.ToRGBA(j)))
}
