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
	"math"
	"math/rand"
)

type lookupPref int

const (
	xPref lookupPref = iota
	yPref
	xyPref
	yxPref
)

type lookupPick int

const (
	randomPick   lookupPick = 1
	closestPick             = 2
	farthestPick            = 4
	darkestPick             = 8
	lightestPick            = 16
)

func invert(l lookup2D, pref lookupPref, pick lookupPick) *tableLookup2D {
	inverterList := map[point][]point{}
	x0, y0, x1, y1 := l.Range()
	var u0, u1, v0, v1 int
	first := true
	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			u, v, ok := l.Lookup(x, y)
			if !ok {
				continue
			}
			p := point{u, v}
			inverterList[p] = append(inverterList[p], point{x, y})
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
	inverter := make(map[point]point, len(inverterList))
	for uv, xys := range inverterList {
		bestScore := math.MaxInt
		candidates := make([]point, 0, len(xys))
		for _, p := range xys {
			score := 0
			if pick&(closestPick|farthestPick) != 0 {
				s := 2 * (p.x - p.y)
				if s < 0 {
					s = -s
					s-- // As arbitrary tie breaker, consider x<y "closer".
				}
				if pick&farthestPick != 0 {
					s = -s
				}
				score += s
			}
			if pick&(darkestPick|lightestPick) != 0 {
				s := p.x*256 + p.y
				if s < 0 {
					s = -s
				}
				if pick&lightestPick != 0 {
					s = -s
				}
				score += s * 1024
			}
			if score < bestScore {
				bestScore = score
				candidates = append(candidates[:0], p)
			} else if score == bestScore {
				candidates = append(candidates, p)
			}
		}
		if pick&randomPick != 0 {
			inverter[uv] = candidates[rand.Intn(len(candidates))]
		} else {
			if len(candidates) != 1 {
				log.Fatalf("random picking not allowed and more than one candidate remaining: %v", candidates)
			}
			inverter[uv] = candidates[0]
		}
	}
	needX, needY := true, true
	stepX, stepY := 1, 1
	for i := 0; needX || needY; i++ {
		var spreadX bool
		switch pref {
		case xPref:
			spreadX = !needY
		case yPref:
			spreadX = needX
		case xyPref:
			spreadX = i%2 == 1
		case yxPref:
			spreadX = i%2 == 0
		}
		if spreadX {
			if !needX {
				continue
			}
			needX = false
			for uv, xy := range inverter {
				uvm := point{uv.x - stepX, uv.y}
				if uvm.x >= u0 {
					if _, ok := inverter[uvm]; !ok {
						inverter[uvm] = xy
						needX = true
					}
				}
				uvp := point{uv.x + stepX, uv.y}
				if uvp.x <= u1 {
					if _, ok := inverter[uvp]; !ok {
						inverter[uvp] = xy
						needX = true
					}
				}
			}
			stepX *= 2
			if stepX >= t.sx {
				needX = false
			}
		} else {
			if !needY {
				continue
			}
			needY = false
			for uv, xy := range inverter {
				uvm := point{uv.x, uv.y - stepY}
				if uvm.y >= v0 {
					if _, ok := inverter[uvm]; !ok {
						inverter[uvm] = xy
						needY = true
					}
				}
				uvp := point{uv.x, uv.y + stepY}
				if uvp.y <= v1 {
					if _, ok := inverter[uvp]; !ok {
						inverter[uvp] = xy
						needY = true
					}
				}
			}
			stepY *= 2
			if stepY >= t.sy {
				needY = false
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
