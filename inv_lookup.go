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

type lookupPref int

const (
	xPref lookupPref = iota
	yPref
	xyPref
	yxPref
)

func invert(l lookup2D, pref lookupPref) *tableLookup2D {
	inverter := map[point]point{}
	x0, y0, x1, y1 := l.Range()
	var u0, u1, v0, v1 int
	first := true
	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			u, v, ok := l.Lookup(x, y)
			if !ok {
				continue
			}
			inverter[point{u, v}] = point{x, y}
			if first || u < u0 {
				u0 = u
			}
			if first || u > u1 {
				u1 = u
			}
			if first || v < v0 {
				v0 = v
			}
			if first || v > v1 {
				v1 = v
			}
			first = false
		}
	}
	t := &tableLookup2D{
		x0: u0,
		sx: u1 - u0 + 1,
		y0: v0,
		sy: v1 - v0 + 1,
	}
	needX, needY := true, true
	for i := 0; i < t.sx+t.sy; i++ {
		var spreadX bool
		switch pref {
		case xPref:
			spreadX = i >= t.sy
		case yPref:
			spreadX = i < t.sx
		case xyPref:
			spreadX = i%2 == 1 && i < 2*t.sx || i >= 2*t.sy
		case yxPref:
			spreadX = i%2 == 0 && i < 2*t.sx || i >= 2*t.sy
		}
		if spreadX {
			if !needX {
				continue
			}
			needX = false
			for uv, xy := range inverter {
				if uv.x > u0 {
					uvm := point{uv.x - 1, uv.y}
					if _, ok := inverter[uvm]; !ok {
						inverter[uvm] = xy
						needX = true
					}
				}
				if uv.x < u1 {
					uvp := point{uv.x + 1, uv.y}
					if _, ok := inverter[uvp]; !ok {
						inverter[uvp] = xy
						needX = true
					}
				}
			}
		} else {
			if !needY {
				continue
			}
			needY = false
			for uv, xy := range inverter {
				if uv.y > v0 {
					uvm := point{uv.x, uv.y - 1}
					if _, ok := inverter[uvm]; !ok {
						inverter[uvm] = xy
						needY = true
					}
				}
				if uv.y < v1 {
					uvp := point{uv.x, uv.y + 1}
					if _, ok := inverter[uvp]; !ok {
						inverter[uvp] = xy
						needY = true
					}
				}
			}
		}
		/*
			for v := 0; v < 256; v++ {
				l := ""
				for u := 0; u < 256; u++ {
					uv := point{u, v}
					if _, ok := inverter[uv]; ok {
						l += "*"
					} else {
						l += "."
					}
				}
				log.Printf("%3d: %s", v, l)
			}
		*/
	}
	t.makeData()
	for uv, xy := range inverter {
		t.setAt(uv.x, uv.y, xy.x, xy.y)
	}
	return t
}
