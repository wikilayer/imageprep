# imageprep

[![Tests](https://github.com/wikilayer/imageprep/actions/workflows/tests.yml/badge.svg)](https://github.com/wikilayer/imageprep/actions/workflows/tests.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/wikilayer/imageprep.svg)](https://pkg.go.dev/github.com/wikilayer/imageprep)

Prepare untrusted images with bounded decoding, EXIF orientation correction,
resizing, square cropping, and JPEG or PNG output.

The Go implementation is the leading implementation.

## Development

`make install-tools` installs the linters. `make build` runs lint, compilation,
and the test suite. `make format` formats Go sources.

## Lines of code

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/wikilayer/imageprep/main/.github/loc-history-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="https://raw.githubusercontent.com/wikilayer/imageprep/main/.github/loc-history-light.svg">
  <img alt="Lines of code over time" src="https://raw.githubusercontent.com/wikilayer/imageprep/main/.github/loc-history.svg">
</picture>
