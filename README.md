# `go-xevd`

[![License](https://img.shields.io/github/license/octu0/go-xevd)](https://github.com/octu0/go-xevd/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/octu0/go-xevd?status.svg)](https://godoc.org/github.com/octu0/go-xevd)
[![Go Report Card](https://goreportcard.com/badge/github.com/octu0/go-xevd)](https://goreportcard.com/report/github.com/octu0/go-xevd)
[![Releases](https://img.shields.io/github/v/release/octu0/go-xevd)](https://github.com/octu0/go-xevd/releases)

Go bindings for [mpeg5/xevd](https://github.com/mpeg5/xevd)  
MPEG-5 EVC decoder.

## Requirements

requires xevd [install](https://github.com/mpeg5/xevd#how-to-build) on your system

```
$ git clone https://github.com/mpeg5/xevd.git
$ cd xevd
$ mkdir build
$ cd build
$ cmake .. -DSET_PROF=BASE
$ make
$ make install
```

## Usage

### Decode

```go
import "github.com/octu0/go-xevd"

func main() {
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
		// => Frame:IDR Slice:I color:YCbCr420

		if err := saveImage(buf.Img); err != nil {
      panic(err)
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
```
