# keyboard-typing

Small Cardputer ADV keyboard example using `WithKeyboard`, `OnKey`, `KeyDown`,
printable `KeyEvent.Rune`, and Backspace to update a typed-text `Viewport`.
The audio controller is registered so Fn volume and mute shortcuts are consumed
by ModGadget rather than becoming typed input.
Typed text is escaped before it is passed to markup, so `<` is displayed
literally and does not start a markup tag.

```sh
tinygo build -target <cardputer-adv-target> ./examples/keyboard-typing
```

Flashing is a separate operation and is not part of the build.
