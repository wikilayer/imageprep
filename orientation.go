package imageprep

import (
	"bytes"
	"errors"
	"fmt"
	"image"

	"github.com/bep/imagemeta"
)

func exifOrientation(mime string, buf []byte) (int, error) {
	const upright = 1
	var format imagemeta.ImageFormat
	switch mime {
	case "image/jpeg":
		format = imagemeta.JPEG
	case "image/png":
		format = imagemeta.PNG
	case "image/webp":
		format = imagemeta.WebP
	default:
		return upright, nil
	}
	o := upright
	var warning error
	_, err := imagemeta.Decode(imagemeta.Options{
		R: bytes.NewReader(buf), ImageFormat: format, Sources: imagemeta.EXIF,
		Warnf: func(format string, args ...any) {
			if warning == nil {
				warning = fmt.Errorf(format, args...)
			}
		},
		HandleTag: func(ti imagemeta.TagInfo) error {
			if ti.Tag != "Orientation" {
				return nil
			}
			if v, ok := asInt(ti.Value); ok && v >= 1 && v <= 8 {
				o = v
			}
			return imagemeta.ErrStopWalking
		},
	})
	err = errors.Join(err, warning)
	if err != nil {
		return upright, fmt.Errorf("read the EXIF of a %s: %w", mime, err)
	}
	return o, nil
}

func asInt(v any) (int, bool) {
	switch n := v.(type) {
	case uint8:
		return int(n), true
	case uint16:
		return int(n), true
	case uint32:
		return int(n), true
	case uint64:
		return int(n), true
	case int:
		return n, true
	case int16:
		return int(n), true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	}
	return 0, false
}

func orientUpright(img image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.At(b.Min.X+x, b.Min.Y+y)
			var dx, dy int
			switch o {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, c)
		}
	}
	return dst
}
