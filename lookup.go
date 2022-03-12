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

type lookup2D interface {
	Range() (x0, x1, y0, y1 int)
	Lookup(x, y int) (u, v int, ok bool)
}

type point struct {
	x, y int
}

type tableLookup2D struct {
	x0, sx, y0, sy int
	data           []point
}

func (t *tableLookup2D) Range() (int, int, int, int) {
	return t.x0, t.x0 + t.sx - 1, t.y0, t.y0 + t.sy - 1
}

func (t *tableLookup2D) Lookup(x, y int) (u, v int, ok bool) {
	if x < t.x0 {
		return 0, 0, false
	}
	rx := x - t.x0
	if rx >= t.sx {
		return 0, 0, false
	}
	if y < t.y0 {
		return 0, 0, false
	}
	ry := y - t.y0
	if ry >= t.sy {
		return 0, 0, false
	}
	p := t.data[rx+t.sx*ry]
	return p.x, p.y, true
}

func (t *tableLookup2D) makeData() {
	t.data = make([]point, t.sx*t.sy)
}

func (t *tableLookup2D) setAt(x, y, u, v int) {
	rx := x - t.x0
	ry := y - t.y0
	t.data[rx+t.sx*ry] = point{u, v}
}
