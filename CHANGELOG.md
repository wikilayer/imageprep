# Changelog

## 0.1.1 — 2026-11-26

- An EXIF entry of an unknown type is now skipped instead of stopping the read, so a valid Orientation entry in the same EXIF block still turns the image upright. In 0.1.0 such an entry made the whole block unreadable: the image stayed unrotated and `MetadataError` was set.
- An Orientation entry of an unknown type is skipped the same way: the image is not rotated and `MetadataError` stays nil. `MetadataError` is still set when the EXIF reader reports an error or a warning.

## 0.1.0 — 2026-10-16

- Prepare JPEG, PNG, GIF, and WebP images with explicit byte and pixel budgets.
- Correct EXIF orientation, fit images without enlargement, or crop and resize a centred square.
- Encode JPEG or PNG without source metadata and report EXIF diagnostics to the caller.
