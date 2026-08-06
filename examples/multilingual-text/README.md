# multilingual-text

Cardputer ADV example for multilingual text and markup through the public
ModGadget API. One screen shows English, Japanese, Chinese, and Korean alongside
named `Style` entries, foreground/background colors, `<b>`, and `<br>`.

Main APIs: `New`, `WithStyles`, `Viewport`, `SetText`, and `Render`.

```sh
tinygo build -target <cardputer-adv-target> ./examples/multilingual-text
```

Flashing is a separate operation and is not performed by this build command.
