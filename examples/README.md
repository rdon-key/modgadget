# Examples

All examples target Cardputer ADV and are built with TinyGo.

| example | purpose | hardware | main APIs |
| --- | --- | --- | --- |
| `hello-text` | Minimal text output | Cardputer ADV | `New`, `WithStyles`, `Viewport`, `SetText`, `Render` |
| `multilingual-text` | Multilingual Style, markup, and Bold | Cardputer ADV | `StyleSet`, `SetText`, `<style>`, `<b>`, `<br>` |
| `text-ticker` | Static, one-shot, and loop scrolling | Cardputer ADV | `SetHorizontalScroll`, `ScrollSpeed`, `ScrollGap`, `ScrollLoop`, `ScrollFromRight` |
| `cardputer-adv-audio` | Audio patterns and system volume diagnostics | Cardputer ADV | `WithVolumeController`, `WithKeyboard`, `Gadget.Update` |

Run `python tools/build_examples.py` to build all four examples without leaving
binaries in the repository. Use `python tools/build_examples.py --help` to see
the available options. Flashing is always a separate operation.

Practical examples that combine multiple APIs are maintained in
[`modgadget-examples`](https://github.com/rdon-key/modgadget-examples). The
complete Rdon Type 100 application is maintained separately in
[`rdon-type100`](https://github.com/rdon-key/rdon-type100).
