# Examples

All examples target Cardputer ADV and are built with TinyGo.

| example | purpose | hardware | main APIs |
| --- | --- | --- | --- |
| `hello-text` | Minimal text output | Cardputer ADV | `New`, `WithStyles`, `Viewport`, `SetText`, `Render` |
| `multilingual-text` | Multilingual Style, markup, and Bold | Cardputer ADV | `StyleSet`, `SetText`, `<style>`, `<b>`, `<br>` |
| `text-ticker` | Static, one-shot, and loop scrolling | Cardputer ADV | `SetHorizontalScroll`, `ScrollSpeed`, `ScrollGap`, `ScrollLoop`, `ScrollFromRight` |
| `keyboard-typing` | Direct keyboard events and typed text | Cardputer ADV | `WithKeyboard`, `OnKey`, `KeyEvent`, `Gadget.Update` |
| `cardputer-adv-audio` | Audio patterns and system volume diagnostics | Cardputer ADV | `WithVolumeController`, `WithKeyboard`, `Gadget.Update` |
| `rdon-type100` | Complete multilingual typing application | Cardputer ADV | Display, text, scroll, keyboard, audio, and system controls |

Run `python tools/build_examples.py --help` to build all six examples without
leaving binaries in the repository. Flashing is always a separate operation.
