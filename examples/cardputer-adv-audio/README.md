# cardputer-adv-audio

Cardputer ADV audio diagnostic using the internal cooperative audio player. It
plays Startup, Click, Correct, and Wrong patterns while continuing keyboard and
player updates. ModGadget's system shortcuts provide Fn+= volume up, Fn+- volume
down, and Fn+M mute toggle; this diagnostic prints resulting volume changes.

Main APIs: `WithKeyboard`, `WithVolumeController`, and `Gadget.Update`. The
audio `Player.Update` call remains in the same cooperative main loop.

```sh
tinygo build -target <cardputer-adv-target> ./examples/cardputer-adv-audio
```

Flashing is a separate operation. This is an experimental diagnostic, not a
public ModGadget Audio API.
