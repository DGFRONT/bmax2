package media

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func encode(t *testing.T, w, h int, asPNG bool) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 0, 255})
		}
	}
	var buf bytes.Buffer
	var err error
	if asPNG {
		err = png.Encode(&buf, img)
	} else {
		err = jpeg.Encode(&buf, img, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestPrepare(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		wantSide int
	}{
		{"wide.jpg", encode(t, 800, 400, false), AvatarSize},
		{"tall.png", encode(t, 300, 900, true), AvatarSize},
		{"small.jpg", encode(t, 100, 120, false), 100},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := Prepare(tc.name, tc.data)
			if err != nil {
				t.Fatal(err)
			}
			cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Width != tc.wantSide || cfg.Height != tc.wantSide {
				t.Fatalf("got %dx%d, want %dx%[3]d", cfg.Width, cfg.Height, tc.wantSide)
			}
		})
	}
}

func TestPrepareRejects(t *testing.T) {
	if _, err := Prepare("photo.png", encode(t, 10, 10, false)); err == nil {
		t.Fatal("jpeg named .png was accepted")
	}
	if _, err := Prepare("photo.gif", encode(t, 10, 10, false)); err == nil {
		t.Fatal(".gif was accepted")
	}
	if _, err := Prepare("photo.jpg", []byte("not an image")); err == nil {
		t.Fatal("garbage was accepted")
	}
}
