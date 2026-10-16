package imageprep

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/image/tiff"
)

func options() Options {
	return Options{MaxBytes: 20 << 20, MaxPixels: 50_000_000, Size: 1600, Mode: Fit, Format: JPEG, JPEGQuality: 85}
}

func jpegBlob(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}))
	return buf.Bytes()
}

func TestPrepareFitsWithoutCroppingOrEnlarging(t *testing.T) {
	for _, c := range []struct{ w, h, wantW, wantH int }{
		{3000, 2000, 1600, 1066}, {2000, 3000, 1066, 1600}, {400, 300, 400, 300}, {3000, 1, 1600, 1},
	} {
		out, err := Prepare(bytes.NewReader(jpegBlob(t, c.w, c.h)), options())
		require.NoError(t, err)
		cfg, format, err := image.DecodeConfig(bytes.NewReader(out.Bytes))
		require.NoError(t, err)
		assert.Equal(t, "jpeg", format)
		assert.Equal(t, "image/jpeg", out.ContentType)
		assert.Equal(t, c.wantW, cfg.Width)
		assert.Equal(t, c.wantH, cfg.Height)
		assert.Equal(t, cfg.Width, out.Width)
		assert.Equal(t, cfg.Height, out.Height)
	}
}

func TestPrepareAcceptsEachSupportedFormat(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	var pngBytes, gifBytes bytes.Buffer
	require.NoError(t, png.Encode(&pngBytes, img))
	require.NoError(t, gif.Encode(&gifBytes, img, nil))
	webp, err := base64.StdEncoding.DecodeString("UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA")
	require.NoError(t, err)
	for mime, blob := range map[string][]byte{
		"image/jpeg": jpegBlob(t, 4, 3), "image/png": pngBytes.Bytes(),
		"image/gif": gifBytes.Bytes(), "image/webp": webp,
	} {
		t.Run(mime, func(t *testing.T) {
			assert.True(t, Supports(mime))
			out, err := Prepare(bytes.NewReader(blob), options())
			require.NoError(t, err)
			assert.Positive(t, out.Width)
			assert.Positive(t, out.Height)
			_, err = jpeg.Decode(bytes.NewReader(out.Bytes))
			require.NoError(t, err)
		})
	}
}

func TestPrepareRejectsRegisteredButUnsupportedDecoders(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, tiff.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 8, 8)), nil))
	_, err := Prepare(&buf, options())
	require.ErrorContains(t, err, "unsupported image format")
	assert.False(t, Supports("image/tiff"))
}

func TestPrepareUsesTheFirstGIFFrame(t *testing.T) {
	palette := color.Palette{color.RGBA{R: 255, A: 255}, color.RGBA{B: 255, A: 255}}
	first := image.NewPaletted(image.Rect(0, 0, 2, 2), palette)
	second := image.NewPaletted(image.Rect(0, 0, 2, 2), palette)
	for i := range second.Pix {
		second.Pix[i] = 1
	}
	var input bytes.Buffer
	require.NoError(t, gif.EncodeAll(&input, &gif.GIF{
		Image: []*image.Paletted{first, second}, Delay: []int{1, 1},
	}))
	o := options()
	o.Format = PNG
	out, err := Prepare(&input, o)
	require.NoError(t, err)
	decoded, err := png.Decode(bytes.NewReader(out.Bytes))
	require.NoError(t, err)
	assert.Equal(t, color.RGBA{R: 255, A: 255}, color.RGBAModel.Convert(decoded.At(0, 0)))
}

func onePixelPNGDeclaring(t *testing.T, w, h uint32) []byte {
	t.Helper()
	var buf, ihdr bytes.Buffer
	buf.WriteString("\x89PNG\r\n\x1a\n")
	ihdr.WriteString("IHDR")
	require.NoError(t, binary.Write(&ihdr, binary.BigEndian, w))
	require.NoError(t, binary.Write(&ihdr, binary.BigEndian, h))
	ihdr.Write([]byte{8, 0, 0, 0, 0})
	require.NoError(t, binary.Write(&buf, binary.BigEndian, uint32(ihdr.Len()-4)))
	buf.Write(ihdr.Bytes())
	require.NoError(t, binary.Write(&buf, binary.BigEndian, crc32.ChecksumIEEE(ihdr.Bytes())))
	return buf.Bytes()
}

func TestPrepareChecksPixelsBeforeDecoding(t *testing.T) {
	_, err := Prepare(bytes.NewReader(onePixelPNGDeclaring(t, 60000, 60000)), options())
	require.ErrorContains(t, err, "pixel budget")
}

func TestPrepareBoundsInputAndAcceptsExactlyTheBudget(t *testing.T) {
	blob := jpegBlob(t, 8, 8)
	o := options()
	o.MaxBytes = int64(len(blob))
	_, err := Prepare(bytes.NewReader(blob), o)
	require.NoError(t, err)
	r := strings.NewReader(string(blob) + strings.Repeat("x", 100))
	_, err = Prepare(r, o)
	require.ErrorContains(t, err, "byte budget")
	assert.Equal(t, 99, r.Len(), "only one excess byte is consumed")
}

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

func TestPreparePropagatesReadFailures(t *testing.T) {
	broken := errors.New("read broke")
	_, err := Prepare(failingReader{broken}, options())
	assert.ErrorIs(t, err, broken)
	blob := jpegBlob(t, 8, 8)
	o := options()
	o.MaxBytes = int64(len(blob))
	_, err = Prepare(io.MultiReader(bytes.NewReader(blob), failingReader{broken}), o)
	assert.ErrorIs(t, err, broken)
}

func TestPrepareRejectsInvalidOptionsBeforeReading(t *testing.T) {
	for _, change := range []func(*Options){
		func(o *Options) { o.MaxBytes = 0 }, func(o *Options) { o.MaxPixels = 0 },
		func(o *Options) { o.Size = 0 }, func(o *Options) { o.Mode = Mode(99) },
		func(o *Options) { o.Format = "image/gif" }, func(o *Options) { o.JPEGQuality = 0 },
		func(o *Options) { o.JPEGQuality = 101 },
		func(o *Options) { o.Mode, o.Size = Square, 1<<30 },
	} {
		o := options()
		change(&o)
		_, err := Prepare(failingReader{errors.New("should not read")}, o)
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "should not read")
	}
}

func TestPrepareSquareEncodesEachOutputFormatAtTheRequestedSize(t *testing.T) {
	for _, shape := range []image.Point{{1200, 300}, {300, 1200}, {40, 40}} {
		for _, format := range []Format{PNG, JPEG} {
			o := options()
			o.Mode, o.Size, o.Format = Square, 320, format
			out, err := Prepare(bytes.NewReader(jpegBlob(t, shape.X, shape.Y)), o)
			require.NoError(t, err)
			cfg, _, err := image.DecodeConfig(bytes.NewReader(out.Bytes))
			require.NoError(t, err)
			assert.Equal(t, 320, cfg.Width)
			assert.Equal(t, 320, cfg.Height)
			assert.Equal(t, string(format), out.ContentType)
		}
	}
}

func TestSquareCropsTheCentreAndKeepsTransparency(t *testing.T) {
	for _, bounds := range []image.Rectangle{image.Rect(0, 0, 6, 2), image.Rect(0, 0, 2, 6)} {
		img := image.NewNRGBA(bounds)
		for y := range bounds.Dy() {
			for x := range bounds.Dx() {
				img.SetNRGBA(x, y, color.NRGBA{R: 255, A: 255})
			}
		}
		for y := range 2 {
			for x := range 2 {
				img.SetNRGBA(x+(bounds.Dx()-2)/2, y+(bounds.Dy()-2)/2, color.NRGBA{G: 200, A: 128})
			}
		}
		var input bytes.Buffer
		require.NoError(t, png.Encode(&input, img))
		o := options()
		o.Mode, o.Size, o.Format = Square, 2, PNG
		out, err := Prepare(&input, o)
		require.NoError(t, err)
		decoded, err := png.Decode(bytes.NewReader(out.Bytes))
		require.NoError(t, err)
		for y := range 2 {
			for x := range 2 {
				r, g, _, a := decoded.At(x, y).RGBA()
				assert.Zero(t, r)
				assert.Positive(t, g)
				assert.Equal(t, uint32(128*257), a)
			}
		}
	}
}

func TestFitSize(t *testing.T) {
	for _, c := range []struct{ w, h, cap, wantW, wantH int }{
		{3000, 2000, 1600, 1600, 1066}, {2000, 3000, 1600, 1066, 1600},
		{400, 300, 1600, 400, 300}, {0, 2, 1, 0, 0}, {2, 0, 1, 0, 0}, {2, 2, 0, 0, 0},
		{math.MaxInt, math.MaxInt, math.MaxInt - 1, math.MaxInt - 1, math.MaxInt - 1},
	} {
		w, h := FitSize(c.w, c.h, c.cap)
		assert.Equal(t, c.wantW, w)
		assert.Equal(t, c.wantH, h)
	}
}
