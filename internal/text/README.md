# Internal text engine

This package contains ModGadget's internal glyph, layout, measurement, and
rendering interfaces. Applications use the opaque `modgadget.Font` handle and
the root `Style`, `StyleSet`, and `MeasureText` APIs instead.

The internal Font interface is deliberately not an external extension point.
