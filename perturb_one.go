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
	"math"
	"math/rand"
)

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
