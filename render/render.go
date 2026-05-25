// Package render provides small image helpers for e-paper screens.
package render

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Renderer owns a parsed OpenType font and creates faces from it.
type Renderer struct {
	font *opentype.Font
}

// NewRenderer parses fontData as an OpenType font or TrueType collection.
func NewRenderer(fontData []byte) (*Renderer, error) {
	if col, err := opentype.ParseCollection(fontData); err == nil {
		f, err := col.Font(0)
		if err != nil {
			return nil, fmt.Errorf("load font collection entry 0: %w", err)
		}
		return &Renderer{font: f}, nil
	}

	f, err := opentype.Parse(fontData)
	if err != nil {
		return nil, fmt.Errorf("parse font: %w", err)
	}
	return &Renderer{font: f}, nil
}

// NewFace returns a font.Face for the renderer font at the given point size.
func (r *Renderer) NewFace(size float64) (font.Face, error) {
	return opentype.NewFace(r.font, &opentype.FaceOptions{
		Size: size,
		DPI:  72,
	})
}

// NewPair creates a pair of blank white canvases.
func NewPair(width, height int) (black, red *image.RGBA) {
	rect := image.Rect(0, 0, width, height)
	black = image.NewRGBA(rect)
	red = image.NewRGBA(rect)
	draw.Draw(black, black.Bounds(), image.White, image.Point{}, draw.Src)
	draw.Draw(red, red.Bounds(), image.White, image.Point{}, draw.Src)
	return black, red
}

// DrawText draws text onto img at (x, y), where y is the baseline.
func DrawText(img *image.RGBA, face font.Face, x, y int, text string) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.Black,
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
}

// DrawHLine draws a horizontal black line from x1 to x2 at row y.
func DrawHLine(img *image.RGBA, x1, x2, y int) {
	for x := x1; x <= x2; x++ {
		img.SetRGBA(x, y, color.RGBA{0, 0, 0, 255})
	}
}
