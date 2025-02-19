// Package main provides an app to calculate image sharpness using Laplace edge-detection.
package main

import (
	"flag"
	"fmt"
	"image"
	"log"
	"net/http"
	"os"
	"strings"

	image2 "github.com/vearutop/photo-blog/internal/infra/image"
	"github.com/vearutop/photo-blog/internal/infra/image/sharpness"
)

func main() {
	var (
		topFrac  float64
		saveEdge string
	)

	flag.StringVar(&saveEdge, "save-edge", "", "Save detected edges to this file.")
	flag.Float64Var(&topFrac, "top-frac", 0.01, "Top fraction")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintf(flag.CommandLine.Output(), "Sharpness returns a value from 0 to 255 that indicates how sharp the image seems.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "It uses Laplace edge-detection to identify areas of contrast.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage:\n")
		fmt.Fprintf(flag.CommandLine.Output(), " sharpness <file.jpg>\n")
		flag.PrintDefaults()
		return
	}

	imgSrc := flag.Arg(0)

	var (
		gray *image.Gray
		err  error
	)

	if strings.HasPrefix(imgSrc, "http://") || strings.HasPrefix(imgSrc, "https://") {
		resp, err := http.Get(imgSrc)
		if err != nil {
			log.Fatal(err)
		}

		gray, err = image2.GrayJPEG(resp.Body)

		if err != nil {
			log.Fatal(err)
		}

		resp.Body.Close()
	} else {
		f, err := os.Open(imgSrc)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		gray, err = image2.GrayJPEG(f)
		if err != nil {
			log.Fatal(err)
		}
	}

	var resultEdge *image.Gray
	if saveEdge != "" {
		resultEdge = image.NewGray(gray.Bounds())
	}

	sh, _, err := sharpness.Custom(gray, resultEdge, topFrac)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(sh)

	if saveEdge != "" {
		if err := image2.SaveJPEG(resultEdge, saveEdge); err != nil {
			log.Fatal(err)
		}
	}
}
