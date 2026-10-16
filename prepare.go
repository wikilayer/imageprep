package imageprep

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math/bits"
	"net/http"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// Mode chooses how the decoded image is resized. Fit preserves the aspect ratio
// without enlargement. Square crops the centre and resizes, including enlargement.
type Mode int

const (
	Fit Mode = iota
	Square
)

// Format selects the encoded output.
type Format string

const (
	JPEG Format = "image/jpeg"
	PNG  Format = "image/png"
)

// Options sets input budgets, output dimensions, and encoding.
// MaxBytes, MaxPixels, and Size must be positive. Size is the maximum side for
// Fit and the exact side for Square. JPEGQuality must be 1–100 for JPEG output.
// MaxPixels also bounds the output bitmap. No input or output limits are implicit.
type Options struct {
	MaxBytes    int64
	MaxPixels   int64
	Size        int
	Mode        Mode
	Format      Format
	JPEGQuality int
}

// Result holds the encoded image and its dimensions after orientation and resizing.
// MetadataError reports unreadable EXIF: preparation succeeds with the decoded
// orientation in that case. Callers decide how to report this partial result.
type Result struct {
	Bytes         []byte
	ContentType   string
	Width         int
	Height        int
	MetadataError error
}

// Prepare decodes JPEG, PNG, GIF, or WebP, corrects EXIF orientation, resizes,
// and encodes an image. GIF inputs use their first frame; animated WebP is not
// supported. Metadata is not copied to the output. PNG preserves transparency;
// JPEG composites it on black.
// Input bytes and declared pixels are checked before full decoding, regardless
// of other image decoders registered by the calling program.
func Prepare(r io.Reader, o Options) (Result, error) {
	if err := o.validate(); err != nil {
		return Result{}, err
	}
	blob, err := readBounded(r, o.MaxBytes)
	if err != nil {
		return Result{}, err
	}
	mime := http.DetectContentType(blob)
	if !Supports(mime) {
		return Result{}, fmt.Errorf("unsupported image format %q", mime)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(blob))
	if err != nil {
		return Result{}, fmt.Errorf("decode image header: %w", err)
	}
	if !withinPixels(cfg.Width, cfg.Height, o.MaxPixels) {
		return Result{}, fmt.Errorf("image is %dx%d, over the %d pixel budget", cfg.Width, cfg.Height, o.MaxPixels)
	}
	src, _, err := image.Decode(bytes.NewReader(blob))
	if err != nil {
		return Result{}, fmt.Errorf("decode image: %w", err)
	}
	orientation, metadataErr := exifOrientation(mime, blob)
	src = orientUpright(src, orientation)
	src = resized(src, o)
	var out bytes.Buffer
	switch o.Format {
	case JPEG:
		err = jpeg.Encode(&out, src, &jpeg.Options{Quality: o.JPEGQuality})
	case PNG:
		err = png.Encode(&out, src)
	}
	if err != nil {
		return Result{}, fmt.Errorf("encode %s: %w", o.Format, err)
	}
	return Result{
		Bytes: out.Bytes(), ContentType: string(o.Format),
		Width: src.Bounds().Dx(), Height: src.Bounds().Dy(), MetadataError: metadataErr,
	}, nil
}

func (o Options) validate() error {
	switch {
	case o.MaxBytes <= 0 || o.MaxPixels <= 0 || o.Size <= 0:
		return fmt.Errorf("byte budget, pixel budget, and size must be positive")
	case o.Mode != Fit && o.Mode != Square:
		return fmt.Errorf("unknown resize mode %d", o.Mode)
	case o.Format != JPEG && o.Format != PNG:
		return fmt.Errorf("unknown output format %q", o.Format)
	case o.Format == JPEG && (o.JPEGQuality < 1 || o.JPEGQuality > 100):
		return fmt.Errorf("JPEG quality must be between 1 and 100")
	case o.Mode == Square && !withinPixels(o.Size, o.Size, o.MaxPixels):
		return fmt.Errorf("square output exceeds the %d pixel budget", o.MaxPixels)
	}
	return nil
}

func withinPixels(w, h int, budget int64) bool {
	return w > 0 && h > 0 && int64(w) <= budget/int64(h)
}

func readBounded(r io.Reader, limit int64) ([]byte, error) {
	limited := &io.LimitedReader{R: r, N: limit}
	blob, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}
	if limited.N == 0 {
		var extra [1]byte
		n, err := io.ReadFull(r, extra[:])
		if n > 0 {
			return nil, fmt.Errorf("image exceeds the %d byte budget", limit)
		}
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("read image: %w", err)
		}
	}
	return blob, nil
}

// Supports reports whether Prepare accepts this exact MIME type.
func Supports(mime string) bool {
	switch mime {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	}
	return false
}

// FitSize preserves the aspect ratio within maxSide, without enlargement.
// It returns zero dimensions if any argument is non-positive.
func FitSize(w, h, maxSide int) (int, int) {
	if w <= 0 || h <= 0 || maxSide <= 0 {
		return 0, 0
	}
	if w <= maxSide && h <= maxSide {
		return w, h
	}
	if w >= h {
		return maxSide, fitFraction(h, maxSide, w)
	}
	return fitFraction(w, maxSide, h), maxSide
}

func fitFraction(side, maxSide, longest int) int {
	hi, lo := bits.Mul64(uint64(side), uint64(maxSide))
	q, _ := bits.Div64(hi, lo, uint64(longest))
	return max(1, int(q))
}

func resized(src image.Image, o Options) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if o.Mode == Square {
		side := min(w, h)
		x, y := b.Min.X+(w-side)/2, b.Min.Y+(h-side)/2
		b = image.Rect(x, y, x+side, y+side)
		w, h = o.Size, o.Size
	} else {
		w, h = FitSize(w, h, o.Size)
		if w == b.Dx() && h == b.Dy() {
			return src
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}
