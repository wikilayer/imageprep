# imageprep

[![Tests](https://github.com/wikilayer/imageprep/actions/workflows/tests.yml/badge.svg)](https://github.com/wikilayer/imageprep/actions/workflows/tests.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/wikilayer/imageprep.svg)](https://pkg.go.dev/github.com/wikilayer/imageprep)

Prepare untrusted images with bounded decoding, EXIF orientation correction,
resizing, square cropping, and JPEG or PNG output.

The Go implementation is the leading implementation.

## Use

```sh
go get github.com/wikilayer/imageprep@v0.1.0
```

```go
result, err := imageprep.Prepare(source, imageprep.Options{
    MaxBytes:    15 << 20,
    MaxPixels:   50_000_000,
    Size:        1600,
    Mode:        imageprep.Fit,
    Format:      imageprep.JPEG,
    JPEGQuality: 85,
})
if err != nil {
    return err
}
if result.MetadataError != nil {
    slog.Warn("image metadata could not be read", "error", result.MetadataError)
}
```

`source` is an `io.Reader`. `result.Bytes` holds the encoded image;
`ContentType`, `Width`, and `Height` describe that output.

JPEG, PNG, GIF, and static WebP inputs are supported. GIF uses its first frame;
animated WebP is not supported. Byte and declared-pixel limits are checked before decoding;
registering another Go image decoder does not expand the accepted formats.
`Supports` reports whether an exact MIME type is accepted.

`Fit` preserves the aspect ratio within `Size` and never enlarges the picture.
`Square` crops the centre and produces exactly `Size` by `Size` pixels, enlarging
small inputs when needed. The pixel budget also bounds a square output.
`FitSize` calculates fit dimensions without decoding an image.

All budgets and dimensions must be positive. JPEG quality must be 1–100 when
writing JPEG; PNG ignores that field. There are no implicit input limits.

EXIF orientations 1–8 are applied before resizing. Metadata is not copied into
the output. EXIF decoder errors and warnings are returned as `MetadataError`
alongside a successful result in its decoded orientation. They do not turn an
otherwise readable picture into a failed preparation. PNG preserves transparency;
JPEG composites transparent pixels on black.

The package does not download, authorize, or store images. Callers choose their
budgets, dimensions, output format, and how to report metadata problems.
HEIC is not currently supported.

See the [Go reference](https://pkg.go.dev/github.com/wikilayer/imageprep) for the
complete API and [CHANGELOG](https://github.com/wikilayer/imageprep/blob/main/CHANGELOG.md)
for releases.

## Development

`make install-tools` installs the linters. `make build` runs lint, compilation,
and the test suite. `make format` formats Go sources.

## Lines of code

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/wikilayer/imageprep/main/.github/loc-history-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/wikilayer/imageprep/main/.github/loc-history-light.svg">
  <img alt="Lines of code over time" src="https://raw.githubusercontent.com/wikilayer/imageprep/main/.github/loc-history.svg">
</picture>
