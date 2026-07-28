# Spleen 8x16 ASCII subset

This package contains a generated subset of Spleen 8x16.

Upstream source:

https://github.com/fcambus/spleen

Upstream version:

2.1.0

The subset contains the 95 printable ASCII characters from U+0020 through
U+007E. `characters.txt` records the input character set used for generation.

The source BDF is maintained separately in:

https://github.com/rdon-key/modgadget-font-assets

Generated with:

    go run ../modgadget-fonts/cmd/modgadget-fonts \
        -bdf ../modgadget-font-assets/fonts/spleen-8x16/spleen-8x16.bdf \
        -subset internal/fontdata/spleen8x16/characters.txt \
        -package spleen8x16 \
        -var Font \
        -o internal/fontdata/spleen8x16/font.go

The generated bitmap data remains subject to the Spleen license stored in
`LICENSES/spleen-BSD-2-Clause.txt`.
