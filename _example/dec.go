package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/octu0/go-xevd"
)

func main() {
	var output bool
	flag.BoolVar(&output, "output", false, "output png")
	flag.Parse()

	decoder, err := xevd.CreateDefaultBaselineDecoder()
	if err != nil {
		panic(err)
	}
	defer decoder.Close()

	f, err := os.Open("./testdata/src.evc")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	br := bufio.NewReader(f)
	if err := decoder.DecodeStream(br, func(buf *xevd.DecodeImageBuffer) {
		defer buf.Close()

		fmt.Printf("Frame:%s Slice:%s color:%s\n", buf.NALUnit, buf.Slice, buf.ColorSpace)

		if output {
			path, err := saveImage(buf.Img)
			if err != nil {
				panic(err)
			}
			fmt.Printf("image saved %s\n", path)
		}
	}); err != nil {
		panic(err)
	}
}

func saveImage(img image.Image) (string, error) {
	out, err := os.CreateTemp("/tmp", "out*.png")
	if err != nil {
		return "", err
	}
	defer out.Close()

	if err := png.Encode(out, img); err != nil {
		return "", err
	}
	return out.Name(), nil
}
