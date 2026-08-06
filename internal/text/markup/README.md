# Internal text markup

This package parses ModGadget's small markup language into internal text spans.
Applications use `Viewport.SetText` to render markup or `modgadget.MeasureText`
to measure it; the parser and span representation are not public APIs.

Supported syntax is `<style=name>`, `</style>`, `<b>`, `</b>`, `<br>`,
`<br/>`, and `<<` for a literal less-than sign. Named styles are complete
styles. An explicit `<b>` overlays Bold while active and closing tags restore
the surrounding style.
