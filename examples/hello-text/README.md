# hello-text

Minimal ModGadget text output for Cardputer ADV. It configures the display,
creates a `Gadget` with `New` and `WithStyles`, creates one `Viewport`, sets
`Hello, ModGadget!` with `SetText`, and renders it once.

Build with TinyGo:

```sh
tinygo build -target=m5stamp-s3a ./examples/hello-text
```

Flashing is a separate operation. Use the target and port appropriate for your
board.

