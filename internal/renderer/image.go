package renderer

import (
	"image"

	"github.com/gdamore/tcell/v2"
)

// Draws an RGBA image using half-block characters with caching
func (r *Renderer) RenderImage(img *image.RGBA, offsetX, offsetY int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if img == nil || r.screen == nil || r.closed {
		return
	}

	bounds := img.Bounds()
	imgW := bounds.Dx()
	imgH := bounds.Dy()

	if imgW <= 0 || imgH <= 0 {
		return
	}

	screenW, screenH := r.screen.Size()
	if screenW <= 0 || screenH <= 0 {
		return
	}
	cellW := imgW
	cellH := (imgH + 1) / 2

	// Manage diff cache
	bufsize := cellW * cellH
	if len(r.prevCells) != bufsize || r.prevW != cellW || r.prevH != cellH {
		r.prevCells = make([]uint64, bufsize)
		r.prevW = cellW
		r.prevH = cellH
		for i := range r.prevCells {
			r.prevCells[i] = 0xFFFFFFFFFFFFFFFF
		}
	}

	pix := img.Pix
	stride := img.Stride
	idx := 0

	for py := 0; py < imgH; py += 2 {
		cellY := offsetY + py/2
		if cellY < 0 || cellY >= screenH {
			idx += cellW
			continue
		}

		topRowOff := py * stride
		botRowOff := topRowOff + stride
		hasBot := py+1 < imgH

		for px := range imgW {
			cellX := offsetX + px
			if cellX < 0 || cellX >= screenW {
				idx++
				continue
			}

			topOff := topRowOff + px*4
			tr, tg, tb := pix[topOff], pix[topOff+1], pix[topOff+2]

			var br, bg, bb byte
			if hasBot {
				botOff := botRowOff + px*4
				br, bg, bb = pix[botOff], pix[botOff+1], pix[botOff+2]
			} else {
				br, bg, bb = tr, tg, tb
			}

			packed := packColors(tr, tg, tb, br, bg, bb)

			if idx < len(r.prevCells) && r.prevCells[idx] == packed {
				idx++
				continue
			}
			if idx < len(r.prevCells) {
				r.prevCells[idx] = packed
			}
			idx++

			style := tcell.StyleDefault.
				Foreground(tcell.NewRGBColor(int32(tr), int32(tg), int32(tb))).
				Background(tcell.NewRGBColor(int32(br), int32(bg), int32(bb)))

			r.screen.SetContent(cellX, cellY, '▀', nil, style)
		}
	}
}

func packColors(tr, tg, tb, br, bg, bb byte) uint64 {
	return uint64(tr)<<40 | uint64(tg)<<32 | uint64(tb)<<24 |
		uint64(br)<<16 | uint64(bg)<<8 | uint64(bb)
}
