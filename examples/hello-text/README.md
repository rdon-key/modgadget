# hello-text

Minimal ModGadget text output for Cardputer ADV. It configures the display,
creates a `Gadget` with `New` and `WithStyles`, creates one `Viewport`, calls
`SetText`, and renders `Hello, ModGadget!` once.

Build with TinyGo:

```sh
tinygo build -target <cardputer-adv-target> ./examples/hello-text
```

Flashing is a separate operation; use the target and port appropriate for your
board.
