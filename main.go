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
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

var (
	inSRGB   = flag.String("in_srgb", "", "sRGB input image")
	inLinear = flag.String("in_linear", "", "linear input image")
	out      = flag.String("out", "", "output image")
	random   = flag.Bool("random", false, "use a random pattern")
	strength = flag.Float64("strength", 1.0, "filter strength")
	goal     = flag.Bool("goal", false, "linear texture is goal color (not delta from white)")
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
	dMax := uint32(math.RoundToEven(*strength * 65535))
	bounds := out.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			sR, sG, sB, sA := sRGB.At(x, y).RGBA()
			lR, lG, lB, lA := linear.At(x, y).RGBA()
			var onR, onG, onB uint32
			oA := (sA*lA + 32767) / 65535
			if oA != 0 {
				snR := (sR*65535 + sA/2) / sA
				snG := (sG*65535 + sA/2) / sA
				snB := (sB*65535 + sA/2) / sA
				lnR := (lR*65535 + lA/2) / lA
				lnG := (lG*65535 + lA/2) / lA
				lnB := (lB*65535 + lA/2) / lA
				onR = perturbOne(x, y, snR, lnR, dMax)
				onG = perturbOne(x, y, snG, lnG, dMax)
				onB = perturbOne(x, y, snB, lnB, dMax)
			}
			oR := clamp(onR)
			oG := clamp(onG)
			oB := clamp(onB)
			out.SetNRGBA64(x, y, color.NRGBA64{R: oR, G: oG, B: oB, A: uint16(oA)})
		}
	}
}

func clamp(x uint32) uint16 {
	if x > 65535 {
		return 65535
	}
	return uint16(x)
}

// TODO(divVerent): Rather use a lookup table? Needs changing to 0..255 color values then.
// TODO(divVerent): Also have a mode that puts more weight on l (to "hide" something that is invisible in thumbnail)?

func perturbOne(x, y int, s, l, dMax uint32) uint32 {
	// Move the target color.
	var t uint32
	if *goal {
		if dMax < s && l < s-dMax {
			t = s - dMax
		} else {
			t = l
		}
	} else {
		d := dMax * (65535 - l) / 65535
		if d < s {
			t = s - d
		} else {
			t = 0
		}
	}

	if t >= s {
		// Filter can't change anything as this op can only darken.
		return s
	}

	// Requirements:
	// - colors a and b.
	// - sRGB-correct scaling of block should yield s.
	// - Linear scaling should get as close as possible to t.
	// I.e. solve:
	//   (a + b) / 2 close to t
	//   (s2l(a) + s2l(b)) / 2 = s2l(s)
	// Solution: binary search for now.

	ss := s2l(s)
	sdMin := 0.0
	sdMax := math.Min(ss, 1-ss)
	a, b := s, s
	for sdMax-sdMin > 1e-8 {
		sd := (sdMax + sdMin) / 2
		sa := ss - sd
		sb := ss + sd
		a = l2s(sa)
		b = l2s(sb)
		if a+b < 2*t {
			sdMax = sd
		} else {
			sdMin = sd
		}
	}

	var r bool
	if *random {
		r = rand.Intn(2) == 1
	} else {
		r = (x^y)&1 == 1
	}
	if r {
		return b
	}
	return a
}

func s2l(x uint32) float64 {
	f := float64(x) / 65535
	if f <= 0.04045 {
		return f / 12.92
	}
	return math.Pow((f+0.055)/1.055, 2.4)
}

func l2sf(x float64) float64 {
	if x <= 0.0031308 {
		return x * 12.92
	}
	return 1.055*math.Pow(x, 1/2.4) - 0.055
}

func l2s(x float64) uint32 {
	f := l2sf(x)
	return uint32(math.RoundToEven(f * 65535))
}
