package sixel

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"strconv"
)

// Encoder encodes an image to sixel format.
type Encoder struct {
	w io.Writer

	// Width is the maximum width to draw to.
	Width int
	// Height is the maximum height to draw to.
	Height int

	// Colors sets the maximum number of sixel color registers. If the value is
	// below 2, then 255 is used. One register is reserved for transparent
	// pixels, so 255 allows a 254 color palette.
	Colors int
}

// NewEncoder returns a new Encoder.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

type bufferedWriter interface {
	io.Writer
	WriteByte(byte) error
	WriteString(string) (int, error)
}

type sixelWriter struct {
	w       io.Writer
	buffer  bufferedWriter
	scratch [20]byte
}

func newSixelWriter(w io.Writer) sixelWriter {
	buffer, _ := w.(bufferedWriter)
	return sixelWriter{w: w, buffer: buffer}
}

func (w *sixelWriter) writeByte(b byte) {
	if w.buffer != nil {
		_ = w.buffer.WriteByte(b)
		return
	}
	_, _ = w.w.Write([]byte{b})
}

func (w *sixelWriter) writeString(s string) {
	if w.buffer != nil {
		_, _ = w.buffer.WriteString(s)
		return
	}
	_, _ = io.WriteString(w.w, s)
}

func (w *sixelWriter) writeInt(n int) {
	_, _ = w.w.Write(strconv.AppendInt(w.scratch[:0], int64(n), 10))
}

func (w *sixelWriter) writeUint(n uint32) {
	_, _ = w.w.Write(strconv.AppendUint(w.scratch[:0], uint64(n), 10))
}

// Encode writes img as a sixel device control string.
func (e *Encoder) Encode(img image.Image) error {
	nc := e.Colors
	if nc < 2 {
		nc = 255
	}

	width, height := img.Bounds().Dx(), img.Bounds().Dy()
	if width == 0 || height == 0 {
		return nil
	}
	if e.Width > 0 && e.Width < width {
		width = e.Width
	}
	if e.Height > 0 && e.Height < height {
		height = e.Height
	}

	paletted, ok := img.(*image.Paletted)
	if ok {
		if len(paletted.Palette) >= nc {
			return fmt.Errorf("sixel: palette has %d colors, maximum is %d", len(paletted.Palette), nc-1)
		}
	} else {
		var err error
		paletted, err = palettedFromImage(img, nc-1)
		if err != nil {
			return err
		}
	}

	out := newSixelWriter(e.w)

	// DECSIXEL Introducer(\033P0;0;8q) + DECGRA ("1;1): Set Raster Attributes
	out.writeString("\x1bP0;0;8q\"1;1")
	for n, v := range paletted.Palette {
		r, g, b, _ := v.RGBA()
		// DECGCI (#): Graphics Color Introducer
		out.writeByte('#')
		out.writeInt(n + 1)
		out.writeString(";2;")
		out.writeUint(r * 100 / 0xFFFF)
		out.writeByte(';')
		out.writeUint(g * 100 / 0xFFFF)
		out.writeByte(';')
		out.writeUint(b * 100 / 0xFFFF)
	}

	bounds := img.Bounds()
	row := make([]byte, width*(len(paletted.Palette)+1))
	used := make([]bool, len(paletted.Palette)+1)
	for z := 0; z < (height+5)/6; z++ {
		// DECGNL (-): Graphics Next Line
		if z > 0 {
			out.writeByte('-')
		}
		clear(row)
		clear(used)
		for p := 0; p < 6; p++ {
			y := z*6 + p
			if y >= height {
				break
			}
			for x := 0; x < width; x++ {
				_, _, _, alpha := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
				if alpha == 0 {
					continue
				}
				idx := int(paletted.ColorIndexAt(bounds.Min.X+x, bounds.Min.Y+y)) + 1
				used[idx] = true
				row[width*idx+x] |= 1 << uint(p)
			}
		}
		firstColor := true
		for n := 1; n < len(used); n++ {
			if !used[n] {
				continue
			}
			// DECGCR ($): Graphics Carriage Return
			if !firstColor {
				out.writeByte('$')
			}
			firstColor = false
			writeColorSelect(&out, n)
			writeSixelRun(&out, row[width*n:width*(n+1)])
		}
	}
	// string terminator(ST)
	out.writeString("\x1b\\")
	return nil
}

func palettedFromImage(img image.Image, maxColors int) (*image.Paletted, error) {
	bounds := img.Bounds()
	palette := make(color.Palette, 0, min(maxColors, 16))
	index := make(map[color.NRGBA]byte)
	paletted := image.NewPaletted(bounds, nil)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r16, g16, b16, a16 := img.At(x, y).RGBA()
			if a16 == 0 {
				continue
			}
			c := color.NRGBA{
				R: uint8(r16 * 0xFF / a16),
				G: uint8(g16 * 0xFF / a16),
				B: uint8(b16 * 0xFF / a16),
				A: uint8(a16 >> 8),
			}
			idx, ok := index[c]
			if !ok {
				if len(palette) >= maxColors {
					return nil, fmt.Errorf("sixel: image has more than %d colors", maxColors)
				}
				idx = byte(len(palette))
				index[c] = idx
				palette = append(palette, c)
			}
			paletted.SetColorIndex(x, y, idx)
		}
	}
	paletted.Palette = palette
	return paletted, nil
}

func writeColorSelect(w *sixelWriter, n int) {
	w.writeByte('#')
	w.writeInt(n)
}

func writeSixelRun(w *sixelWriter, row []byte) {
	// Zero-valued sixel data advances the active position without drawing
	// pixels. Preserve those gaps before and between drawn runs, but omit
	// trailing empty data because it has no visible effect.
	for len(row) > 0 && row[len(row)-1] == 0 {
		row = row[:len(row)-1]
	}

	last := byte(0)
	count := 0
	for _, ch := range row {
		if count != 0 && ch != last {
			writeRepeat(w, count, last)
			count = 0
		}
		last = ch
		count++
	}
	writeRepeat(w, count, last)
}

func writeRepeat(w *sixelWriter, count int, ch byte) {
	if count == 0 {
		return
	}
	ch += '?'
	switch count {
	case 1:
		w.writeByte(ch)
	case 2:
		w.writeByte(ch)
		w.writeByte(ch)
	case 3:
		w.writeByte(ch)
		w.writeByte(ch)
		w.writeByte(ch)
	default:
		w.writeByte('!')
		w.writeInt(count)
		w.writeByte(ch)
	}
}

// Decoder decodes sixel format into an image.
type Decoder struct {
	r io.Reader
}

// NewDecoder returns a new Decoder.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r}
}

// Decode decodes a sixel device control string into img.
func (e *Decoder) Decode(img *image.Image) error {
	data, err := io.ReadAll(e.r)
	if err != nil {
		return err
	}
	start := bytes.IndexByte(data, '\x1b')
	if start == -1 {
		return nil
	}
	if start+1 >= len(data) || data[start+1] != 'P' {
		return errors.New("invalid format: illegal header")
	}
	i := start + 2
	for i < len(data) && data[i] != 'q' {
		i++
	}
	if i == len(data) {
		return errors.New("invalid format: missing sixel introducer")
	}
	i++

	colors := map[uint]color.Color{
		// 16 predefined color registers of VT340
		0:  sixelRGB(0, 0, 0),
		1:  sixelRGB(20, 20, 80),
		2:  sixelRGB(80, 13, 13),
		3:  sixelRGB(20, 80, 20),
		4:  sixelRGB(80, 20, 80),
		5:  sixelRGB(20, 80, 80),
		6:  sixelRGB(80, 80, 20),
		7:  sixelRGB(53, 53, 53),
		8:  sixelRGB(26, 26, 26),
		9:  sixelRGB(33, 33, 60),
		10: sixelRGB(60, 26, 26),
		11: sixelRGB(33, 60, 33),
		12: sixelRGB(60, 33, 60),
		13: sixelRGB(33, 60, 60),
		14: sixelRGB(60, 60, 33),
		15: sixelRGB(80, 80, 80),
	}
	dx, dy := 0, 0
	dw, dh, w, h := 0, 0, 200, 200
	pimg := image.NewNRGBA(image.Rect(0, 0, w, h))
	var cn uint
data:
	for i < len(data) {
		c := data[i]
		i++
		if c == '\r' || c == '\n' || c == '\b' {
			continue
		}
		switch c {
		case '\x1b':
			if i < len(data) && data[i] == '\\' {
				break data
			}
		case '"':
			params := readParams(data, &i)
			if len(params) >= 4 {
				if w < params[2] {
					w = params[2]
				}
				if h < params[3]+6 {
					h = params[3] + 6
				}
				pimg = expandImage(pimg, w, h)
			}
		case '$':
			dx = 0
		case '!':
			nc, ok := readUint(data, &i)
			if !ok || i >= len(data) {
				return errors.New("invalid format: illegal repeating data tokens")
			}
			c = data[i]
			i++
			if c < '?' || c > '~' {
				return fmt.Errorf("invalid format: illegal repeating data tokens '!%d%c'", nc, c)
			}
			pimg = ensureImageSize(pimg, &w, &h, dx+int(nc), dy+6)
			drawSixelRepeat(pimg, dx, dy, int(nc), c-'?', colors[cn], &dh)
			dx += int(nc)
			if dw < dx {
				dw = dx
			}
		case '-':
			dx = 0
			dy += 6
			pimg = ensureImageSize(pimg, &w, &h, dx+1, dy+6)
		case '#':
			nc, ok := readUint(data, &i)
			if !ok {
				return errors.New("invalid format: illegal color specifier")
			}
			if i < len(data) && data[i] == ';' {
				i++
				csys, ok := readUint(data, &i)
				if !ok || i >= len(data) || data[i] != ';' {
					return errors.New("invalid format: illegal color specifier")
				}
				i++
				r, ok := readUint(data, &i)
				if !ok || i >= len(data) || data[i] != ';' {
					return errors.New("invalid format: illegal color specifier")
				}
				i++
				g, ok := readUint(data, &i)
				if !ok || i >= len(data) || data[i] != ';' {
					return errors.New("invalid format: illegal color specifier")
				}
				i++
				b, ok := readUint(data, &i)
				if !ok {
					return errors.New("invalid format: illegal color specifier")
				}

				if csys == 1 {
					colors[nc] = sixelHLS(r, g, b)
				} else {
					colors[nc] = sixelRGB(r, g, b)
				}
			}
			cn = nc
			if _, ok := colors[cn]; !ok {
				return fmt.Errorf("invalid format: undefined color number %d", cn)
			}
		default:
			if c >= '?' && c <= '~' {
				pimg = ensureImageSize(pimg, &w, &h, dx+1, dy+6)
				drawSixelRepeat(pimg, dx, dy, 1, c-'?', colors[cn], &dh)
				dx++
				if dw < dx {
					dw = dx
				}
				break
			}
			return errors.New("invalid format: illegal data tokens")
		}
	}
	rect := image.Rect(0, 0, dw, dh)
	tmp := image.NewNRGBA(rect)
	draw.Draw(tmp, rect, pimg, image.Point{0, 0}, draw.Src)
	*img = tmp
	return nil
}

func readParams(data []byte, i *int) []int {
	params := []int{}
	for *i < len(data) {
		n, _ := readInt(data, i)
		params = append(params, n)
		if *i >= len(data) || data[*i] != ';' {
			break
		}
		*i = *i + 1
	}
	return params
}

func readInt(data []byte, i *int) (int, bool) {
	n, ok := readUint(data, i)
	return int(n), ok
}

func readUint(data []byte, i *int) (uint, bool) {
	start := *i
	var n uint
	for *i < len(data) {
		c := data[*i]
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + uint(c-'0')
		*i = *i + 1
	}
	return n, *i != start
}

func ensureImageSize(pimg *image.NRGBA, w *int, h *int, minW int, minH int) *image.NRGBA {
	for *w < minW {
		*w *= 2
	}
	for *h < minH {
		*h *= 2
	}
	bounds := pimg.Bounds()
	if bounds.Dx() >= *w && bounds.Dy() >= *h {
		return pimg
	}
	return expandImage(pimg, *w, *h)
}

func drawSixelRepeat(pimg *image.NRGBA, dx int, dy int, count int, bits byte, color color.Color, dh *int) {
	m := byte(1)
	for p := 0; p < 6; p++ {
		if bits&m != 0 {
			for q := 0; q < count; q++ {
				pimg.Set(dx+q, dy+p, color)
			}
			if *dh < dy+p+1 {
				*dh = dy + p + 1
			}
		}
		m <<= 1
	}
}

func sixelRGB(r, g, b uint) color.Color {
	return color.NRGBA{uint8(r * 0xFF / 100), uint8(g * 0xFF / 100), uint8(b * 0xFF / 100), 0xFF}
}

func sixelHLS(h, l, s uint) color.Color {
	var r, g, b, max, min float64

	/* https://wikimedia.org/api/rest_v1/media/math/render/svg/17e876f7e3260ea7fed73f69e19c71eb715dd09d */
	/* https://wikimedia.org/api/rest_v1/media/math/render/svg/f6721b57985ad83db3d5b800dc38c9980eedde1d */
	if l > 50 {
		max = float64(l) + float64(s)*(1.0-float64(l)/100.0)
		min = float64(l) - float64(s)*(1.0-float64(l)/100.0)
	} else {
		max = float64(l) + float64(s*l)/100.0
		min = float64(l) - float64(s*l)/100.0
	}

	/* sixel hue color ring is roteted -120 degree from nowdays general one. */
	h = (h + 240) % 360

	/* https://wikimedia.org/api/rest_v1/media/math/render/svg/937e8abdab308a22ff99de24d645ec9e70f1e384 */
	switch h / 60 {
	case 0: /* 0 <= hue < 60 */
		r = max
		g = min + (max-min)*(float64(h)/60.0)
		b = min
	case 1: /* 60 <= hue < 120 */
		r = min + (max-min)*(float64(120-h)/60.0)
		g = max
		b = min
	case 2: /* 120 <= hue < 180 */
		r = min
		g = max
		b = min + (max-min)*(float64(h-120)/60.0)
	case 3: /* 180 <= hue < 240 */
		r = min
		g = min + (max-min)*(float64(240-h)/60.0)
		b = max
	case 4: /* 240 <= hue < 300 */
		r = min + (max-min)*(float64(h-240)/60.0)
		g = min
		b = max
	case 5: /* 300 <= hue < 360 */
		r = max
		g = min
		b = min + (max-min)*(float64(360-h)/60.0)
	default:
	}
	return sixelRGB(uint(r), uint(g), uint(b))
}

func expandImage(pimg *image.NRGBA, w, h int) *image.NRGBA {
	b := pimg.Bounds()
	if w < b.Max.X {
		w = b.Max.X
	}
	if h < b.Max.Y {
		h = b.Max.Y
	}
	tmp := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(tmp, b, pimg, image.Point{0, 0}, draw.Src)
	return tmp
}
