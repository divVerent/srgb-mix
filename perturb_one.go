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

package main

import (
	"log"
	"math/rand"
)

type filterMode int

const (
	mixL filterMode = iota
	mixS
	darkenL
	lightenS
)

type perturber struct {
	l *tableLookup2D
}

func newPerturber(pref lookupPref, pick lookupPick) *perturber {
	forward := &sRGBLookup2D{}
	inverse := invert(forward, pref, pick)
	return &perturber{
		l: inverse,
	}
}

func (p *perturber) perturbOne(x, y int, s, l uint32, strength int, mode filterMode) uint32 {
	// Think in 8bit colors.
	s8 := int((s + 128) / 257)
	l8 := int((l + 128) / 257)

	// Adjust transform strength.
	switch mode {
	case darkenL:
		l8 = s8 - (strength*(255-l8)+127)/255
		if l8 < 0 {
			l8 = 0
		}
	case lightenS:
		s8 = l8 + (strength*s8+127)/255
		if s8 > 255 {
			s8 = 255
		}
	case mixL:
		l8 = (s8*(255-strength) + l8*strength + 127) / 255
	case mixS:
		s8 = (l8*(255-strength) + s8*strength + 127) / 255
	}

	// Get color pair.
	a, b, ok := p.l.Lookup(l8, s8)
	if !ok {
		log.Fatalf("unreachable code: failed lookup for %v, %v", s8, l8)
	}

	var r bool
	if *random {
		r = rand.Intn(2) == 1
	} else {
		r = (x^y)&1 == 1
	}
	if r {
		return uint32(b * 257)
	}
	return uint32(a * 257)
}
