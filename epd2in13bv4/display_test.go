package epd2in13bv4

import (
	"image"
	"image/color"
	"testing"
)

func TestBufferFromImageRotatesLandscapeImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, LandscapeWidth, LandscapeHeight))
	for y := 0; y < LandscapeHeight; y++ {
		for x := 0; x < LandscapeWidth; x++ {
			img.Set(x, y, color.White)
		}
	}
	img.Set(0, 0, color.Black)

	buf := BufferFromImage(img)
	if len(buf) != BufferSize() {
		t.Fatalf("len(buf) = %d, want %d", len(buf), BufferSize())
	}
	if CountInk(buf) != 1 {
		t.Fatalf("CountInk(buf) = %d, want 1", CountInk(buf))
	}

	lineWidth := (Width + 7) / 8
	wantByte := (LandscapeWidth - 1) * lineWidth
	if got := buf[wantByte]; got != 0x7F {
		t.Fatalf("buf[%d] = %#02x, want 0x7f", wantByte, got)
	}
}

func TestBufferFromImageRejectsWrongSize(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, LandscapeWidth-1, LandscapeHeight))
	img.Set(0, 0, color.Black)

	buf := BufferFromImage(img)
	if len(buf) != BufferSize() {
		t.Fatalf("len(buf) = %d, want %d", len(buf), BufferSize())
	}
	if CountInk(buf) != 0 {
		t.Fatalf("CountInk(buf) = %d, want 0", CountInk(buf))
	}
}
