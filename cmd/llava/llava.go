// Package main provides an app to prompt for a remote image from command line.
//
// # Installation
//
// Get Ollama from https://ollama.com/.
// Then install llava with
//
//	ollama run llava:7b
//
// Then you can run llava for remote images.
//
//	go install github.com/vearutop/photo-blog/cmd/llava@latest
//	llava https://vearutop.p1cs.art/thumb/1200w/3bu7hfd29vjyc.jpg
//
// "A vintage Chevrolet Nomad station wagon, painted in a striking turquoise hue with silver
// trim and a white roof, is on display at an outdoor car show. The vehicle, featuring the
// iconic "Chevy bowtie" emblem on its front grille, is parked in front of a venue that appears
// to be a commercial building with a sports bar theme, identifiable by signage and decor.
// Spectators are visible around the car, admiring its classic design and color scheme."
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

func main() {
	var (
		prompt string
		model  string
	)

	flag.StringVar(&prompt, "prompt", "Generate a detailed caption for this image, don't name the places or items unless you're sure.", "prompt")
	flag.StringVar(&model, "model", "llava:7b", "model name")
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		return
	}

	img := flag.Arg(0)

	resp, err := http.Get(img)
	if err != nil {
		log.Fatal(err)
	}

	cont, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	type Req struct {
		Model  string   `json:"model"`
		Prompt string   `json:"prompt"`
		Stream bool     `json:"stream"`
		Images [][]byte `json:"images"`
	}

	r := Req{}

	r.Model = model
	r.Prompt = prompt
	r.Stream = false
	r.Images = append(r.Images, cont)

	body, err := json.Marshal(r)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, "http://localhost:11434/api/generate", bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}

	resp, err = http.DefaultTransport.RoundTrip(req)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	cont, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	type Resp struct {
		Response string `json:"response"`
	}

	re := Resp{}

	if err := json.Unmarshal(cont, &re); err != nil {
		log.Fatal(err)
	}

	fmt.Println(strings.Trim(re.Response, `" \t`))
}
