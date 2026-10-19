// Package media prepares uploaded avatar images: it checks the format,
// crops them to a centered square and scales them down to AvatarSize.
package media

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"path/filepath"

	"golang.org/x/image/draw"
)

// AvatarSize is the side of stored avatars in pixels (2x the largest
// on-screen size of 64 CSS pixels, with margin for 3x screens).
const AvatarSize = 256

// maxPixels guards against decompression bombs.
const maxPixels = 40_000_000

// ContentType returns the content type for a media name, or "".
func ContentType(name string) string {
	switch filepath.Ext(name) {
	case ".jpg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	}
	return ""
}

// Prepare decodes data, which must match the format of name's extension,
// and returns the square, downscaled image re-encoded in the same format.
func Prepare(name string, data []byte) ([]byte, error) {
	want := ContentType(name)
	if want == "" {
		return nil, errors.New("only .jpg and .png images are supported")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("not a valid image: %w", err)
	}
	if "image/"+format != want {
		return nil, fmt.Errorf("file content is %s but the name ends with %s", format, filepath.Ext(name))
	}
	if cfg.Width*cfg.Height > maxPixels {
		return nil, fmt.Errorf("image is too large: %dx%d", cfg.Width, cfg.Height)
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("not a valid image: %w", err)
	}

	b := src.Bounds()
	side := min(b.Dx(), b.Dy())
	crop := image.Rect(0, 0, side, side).Add(image.Pt(b.Min.X+(b.Dx()-side)/2, b.Min.Y+(b.Dy()-side)/2))
	size := min(side, AvatarSize)
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, crop, draw.Src, nil)

	var out bytes.Buffer
	if format == "jpeg" {
		err = jpeg.Encode(&out, dst, &jpeg.Options{Quality: 85})
	} else {
		err = (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&out, dst)
	}
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
