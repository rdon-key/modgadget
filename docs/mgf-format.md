# MGF1 font file format

MGF (ModGadget Font) is the binary bitmap-font format read by ModGadget. MGF1
stores one font subset and an optional region hint in each file.

MGF1 uses little-endian byte order. All offsets are absolute byte offsets from
the start of the file. The format has three contiguous regions:

1. a 36-byte header;
2. an eight-byte entry for every glyph in the glyph index; and
3. one uncompressed glyph record for every index entry.

MGF1 has no compression, byte padding between regions or records, extension
region, checksum, kerning table, or automatic file-sharding mechanism.

## Header

The header is exactly 36 bytes.

| Offset | Size | Type | Field | Required value or meaning |
| ---: | ---: | --- | --- | --- |
| 0 | 3 | byte[3] | Magic | ASCII `MGF` |
| 3 | 1 | uint8 | Version | `1` |
| 4 | 4 | byte[4] | FontID | Four printable ASCII bytes (`0x20..0x7E`) |
| 8 | 4 | byte[4] | SubsetID | Four printable ASCII bytes (`0x20..0x7E`) |
| 12 | 2 | byte[2] | Region | Two printable ASCII bytes (`0x20..0x7E`), or two zero bytes |
| 14 | 2 | uint16 | GlyphCount | Number of glyphs |
| 16 | 1 | uint8 | Ascent | Recommended height above the baseline |
| 17 | 1 | uint8 | Descent | Recommended height below the baseline |
| 18 | 1 | uint8 | LineGap | Recommended additional line spacing |
| 19 | 1 | uint8 | MaxWidth | Maximum glyph bitmap width in the file |
| 20 | 1 | uint8 | MaxHeight | Maximum glyph bitmap height in the file |
| 21 | 1 | uint8 | Flags | `0` |
| 22 | 2 | uint16 | HeaderSize | `36` |
| 24 | 4 | uint32 | IndexOffset | `36` |
| 28 | 4 | uint32 | GlyphDataOffset | `36 + GlyphCount * 8` |
| 32 | 4 | uint32 | FileSize | Exact length of the file |

`FontID`, `SubsetID`, and `Region` are metadata identifiers. Region does not
describe character coverage; the glyph index is authoritative for coverage.
MGF1 does not require these identifiers to be globally unique.

The recommended line height is:

```text
Ascent + Descent + LineGap
```

Character advance is stored per glyph. Additional letter or line spacing is a
layout concern and is not stored in MGF1.

Because `GlyphCount` is a `uint16`, a file can contain at most 65,535 glyphs.

## Glyph index

The index begins at byte 36 and contains exactly `GlyphCount` entries. Each
entry is eight bytes:

| Entry offset | Size | Type | Field | Meaning |
| ---: | ---: | --- | --- | --- |
| 0 | 4 | uint32 | Codepoint | Unicode scalar value |
| 4 | 4 | uint32 | GlyphOffset | Absolute offset of its glyph record |

Code points must be valid Unicode scalar values: `U+0000..U+D7FF` or
`U+E000..U+10FFFF`. Noncharacters are allowed. Entries must be in strictly
increasing code-point order, with no duplicates, so readers can use binary
search.

Glyph offsets must also be strictly increasing. Every offset must be at least
`GlyphDataOffset` and less than `FileSize`. In the fully validated file, the
first offset equals `GlyphDataOffset`, and each later offset equals the end of
the preceding record.

## Glyph records

Each glyph record has a ten-byte header followed immediately by its bitmap.

| Record offset | Size | Type | Field | Meaning |
| ---: | ---: | --- | --- | --- |
| 0 | 1 | uint8 | Width | Bitmap width in pixels |
| 1 | 1 | uint8 | Height | Bitmap height in pixels |
| 2 | 2 | int16 | AdvanceX | Horizontal pen advance |
| 4 | 2 | int16 | BearingX | Horizontal distance from pen to bitmap left edge |
| 6 | 2 | int16 | BearingY | Vertical distance from baseline to bitmap top edge |
| 8 | 2 | uint16 | DataLength | Bitmap length in bytes |
| 10 | N | byte[] | Bitmap | Raw one-bit bitmap data |

Bitmap data is uncompressed and row-major. Rows run top to bottom, pixels run
left to right, and the first pixel in each byte is its most significant bit.
Every row begins on a byte boundary, so:

```text
rowBytes   = (Width + 7) / 8
DataLength = rowBytes * Height
```

If either dimension is zero, `DataLength` is zero. Padding bits after the last
pixel of a row have no defined value.

Records are contiguous and appear in index order. MGF1 permits no gaps,
overlaps, unreferenced records, duplicate offsets, or trailing data. The end of
the final record must equal `FileSize`. `MaxWidth` and `MaxHeight` must equal
the actual maxima across all records. If `GlyphCount` is zero, both maxima are
zero and `GlyphDataOffset` equals `FileSize`.

## Reader validation

The current reader rejects a file unless all of the following hold:

- the magic, version, identifiers, flags, fixed sizes, offsets, and file length
  satisfy the header rules above;
- the index occupies exactly `GlyphCount * 8` bytes and its code points and
  offsets are valid and strictly increasing;
- every glyph record fits in the file, its dimensions do not exceed the header
  maxima, and its `DataLength` exactly matches the raw bitmap dimensions;
- records cover the complete glyph-data region contiguously; and
- the computed maximum dimensions equal the header values.

Unknown versions and nonzero flags are rejected. MGF1 defines no compressed
bitmap flag or forward-compatible extension data.

## Go usage

Applications normally import packaged fonts from
[`modgadget-fonts`](https://github.com/rdon-key/modgadget-fonts). A package that
embeds its own generated MGF can expose it through the root ModGadget API:


Canonical font sources, provenance, generation, validation, and immutable MGF
assets are maintained in [`modgadget-font-assets`](https://github.com/rdon-key/modgadget-font-assets).

```go
package customfont

import (
	_ "embed"

	"github.com/rdon-key/modgadget"
)

//go:embed custom.mgf
var data string

var Font = modgadget.MustOpenMGF(data)
```

Use `modgadget.OpenMGF` when malformed data should be returned as an error, or
`modgadget.MustOpenMGF` for trusted static data where failure should panic.
Both validate the entire MGF file and retain the embedded string; glyph bitmap
data remains internal to ModGadget's opaque `Font` API.
