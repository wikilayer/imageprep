package imageprep

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrientUpright(t *testing.T) {
	src := image.NewRGBA(image.Rect(10, 20, 13, 22))
	for i := range 6 {
		src.Set(10+i%3, 20+i/3, color.RGBA{R: uint8(i + 1), A: 255})
	}
	for i, want := range [][]uint8{
		{1, 2, 3, 4, 5, 6}, {3, 2, 1, 6, 5, 4}, {6, 5, 4, 3, 2, 1}, {4, 5, 6, 1, 2, 3},
		{1, 4, 2, 5, 3, 6}, {4, 1, 5, 2, 6, 3}, {6, 3, 5, 2, 4, 1}, {3, 6, 2, 5, 1, 4},
	} {
		got := orientUpright(src, i+1)
		w, h := 3, 2
		if i >= 4 {
			w, h = 2, 3
		}
		b := got.Bounds()
		assert.Equal(t, w, b.Dx())
		assert.Equal(t, h, b.Dy())
		for j, red := range want {
			assert.Equal(t, color.RGBA{R: red, A: 255}, got.At(b.Min.X+j%w, b.Min.Y+j/w), "orientation %d pixel %d", i+1, j)
		}
	}
}

func withEXIF(t *testing.T, blob, exif []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	out.Write(blob[:2])
	out.Write([]byte{0xff, 0xe1})
	require.NoError(t, binary.Write(&out, binary.BigEndian, uint16(len(exif)+2)))
	out.Write(exif)
	out.Write(blob[2:])
	return out.Bytes()
}

func TestPrepareReadsOrientationAndDoesNotCopyMetadata(t *testing.T) {
	blob := jpegBlob(t, 30, 20)
	for o := uint8(1); o <= 8; o++ {
		exif := []byte{'E', 'x', 'i', 'f', 0, 0, 'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0,
			0x12, 1, 3, 0, 1, 0, 0, 0, o, 0, 0, 0, 0, 0, 0, 0}
		out, err := Prepare(bytes.NewReader(withEXIF(t, blob, exif)), options())
		require.NoError(t, err)
		assert.NoError(t, out.MetadataError)
		w, h := 30, 20
		if o >= 5 {
			w, h = 20, 30
		}
		decoded, err := jpeg.Decode(bytes.NewReader(out.Bytes))
		require.NoError(t, err)
		assert.Equal(t, image.Rect(0, 0, w, h), decoded.Bounds())
		assert.NotContains(t, string(out.Bytes), "Exif")
	}
}

func TestPrepareReturnsMalformedEXIFAsAnExplicitWarning(t *testing.T) {
	exif := []byte{'E', 'x', 'i', 'f', 0, 0, 'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0,
		0x12, 1, 0xff, 0xff, 1, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0}
	blob := withEXIF(t, jpegBlob(t, 30, 20), exif)
	out, err := Prepare(bytes.NewReader(blob), options())
	require.NoError(t, err)
	assert.Error(t, out.MetadataError)
	assert.Equal(t, 30, out.Width)
	assert.Equal(t, 20, out.Height)
}
