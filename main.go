/*
Copyright (c) 2022, Rudolf Polzer
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
   list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

// srgbmix.go - a program to mix two images so that one normally sees one image, but when scaling sees another.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
)

var (
	inSRGB     = flag.String("in_srgb", "", "sRGB input image")
	inLinear   = flag.String("in_linear", "", "linear input image")
	out        = flag.String("out", "", "output image")
	random     = flag.Bool("random", false, "use a random pattern")
	strength   = flag.Float64("strength", 1.0, "filter strength")
	preference = flag.String("preference", "auto", "importance of sRGB vs linear image (auto/s/l/sl/ls)")
	mode       = flag.String("mode", "darken_l", "filter mode: darken_l/lighten_s/ mix_l/mix_s")
)

func loadImage(name string) (image.Image, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("failed to open image %v: %v", name, err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image %v: %v", name, err)
	}
	return img, nil
}

func writeImage(img image.Image, name string) (err error) {
	var encode func(f io.Writer, img image.Image) error
	switch strings.ToLower(filepath.Ext(name)) {
	case ".gif":
		encode = func(f io.Writer, img image.Image) error {
			return gif.Encode(f, img, &gif.Options{
				NumColors: 256,
			})
		}
	case ".jpg", ".jpeg":
		encode = func(f io.Writer, img image.Image) error {
			return jpeg.Encode(f, img, &jpeg.Options{
				Quality: 100,
			})
		}
	case ".png":
		encode = png.Encode
	default:
		return fmt.Errorf("could not find encoder for image file name %v")
	}
	f, err := os.Create(name)
	if err != nil {
		return fmt.Errorf("failed to open image %v: %v", name, err)
	}
	defer func() {
		errC := f.Close()
		if err == nil {
			err = errC
		}
	}()
	return encode(f, img)
}

func main() {
	flag.Parse()
	sRGB, err := loadImage(*inSRGB)
	if err != nil {
		log.Fatalf("failed to load --in_srgb: %v", err)
	}
	linear, err := loadImage(*inLinear)
	if err != nil {
		log.Fatalf("failed to load --in_linear: %v", err)
	}
	bounds := sRGB.Bounds()
	if bounds != linear.Bounds() {
		log.Fatalf("input images must have same bounds; got %v and %v", bounds, linear.Bounds())
	}
	img := image.NewNRGBA64(bounds)
	perturb(sRGB, linear, img)
	err = writeImage(img, *out)
	if err != nil {
		log.Fatalf("failed to write --out: %v", err)
	}
}

func perturb(sRGB, linear image.Image, out *image.NRGBA64) {
	var pref lookupPref
	var mod filterMode
	switch *mode {
	case "darken_l":
		pref = yPref
		mod = darkenL
	case "lighten_s":
		pref = xPref
		mod = lightenS
	case "mix_l":
		pref = yPref
		mod = mixL
	case "mix_s":
		pref = xPref
		mod = mixS
	default:
		log.Fatalf("--mode must be darken_l, lighten_s, mix_l or mix_s")
	}
	switch *preference {
	case "l":
		pref = xPref
	case "s":
		pref = yPref
	case "ls":
		pref = xyPref
	case "sl":
		pref = yxPref
	case "auto":
		// keep
	default:
		log.Fatalf("--preference must be auto, l, s, ls or sl")
	}
	p := newPerturber(pref)
	strength := int(math.RoundToEven(*strength * 255))
	bounds := out.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			sR, sG, sB, sA := sRGB.At(x, y).RGBA()
			lR, lG, lB, lA := linear.At(x, y).RGBA()
			var oR, oG, oB uint32
			oA := (sA*lA + 32767) / 65535
			if oA != 0 {
				snR := (sR*65535 + sA/2) / sA
				snG := (sG*65535 + sA/2) / sA
				snB := (sB*65535 + sA/2) / sA
				lnR := (lR*65535 + lA/2) / lA
				lnG := (lG*65535 + lA/2) / lA
				lnB := (lB*65535 + lA/2) / lA
				oR = p.perturbOne(x, y, snR, lnR, strength, mod)
				oG = p.perturbOne(x, y, snG, lnG, strength, mod)
				oB = p.perturbOne(x, y, snB, lnB, strength, mod)
			}
			out.SetNRGBA64(x, y, color.NRGBA64{R: uint16(oR), G: uint16(oG), B: uint16(oB), A: uint16(oA)})
		}
	}
}
