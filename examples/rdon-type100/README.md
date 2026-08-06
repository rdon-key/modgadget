# Rdon Type 100

Rdon Type 100 is an experimental multilingual typing-game example for the
Cardputer ADV. Japanese, English, Chinese, Korean, and All Languages each
contain 20 playable questions.

## Current flow

1. The display is cleared and a short three-tone startup sound is played using
   the existing two-tone Startup pattern followed by the one-tone Click pattern.
2. `Rdon Type 100` is shown for about 1.5 seconds.
3. The menu offers Japanese, English, Chinese, Korean, and All Languages.
4. Enter starts the selected course.
5. Completing all 20 questions shows the final time and miss count.

The 16-dot title is centered from its measured glyph width. One fixed outer
frame surrounds all five menu rows; individual rows have no boxes or divider
lines. The 24-dot guide scrolls in red along the bottom of the display.

Audio is cooperative and finite. Startup uses the built-in 660/880 Hz Startup
pattern followed once by the 1400 Hz Click pattern. Moving the cursor plays the
100 ms, 1400 Hz Click pattern. Enter plays the distinct two-tone Correct pattern
(1047 Hz for 300 ms, a 100 ms silent gap, then 784 Hz for 260 ms). Every pattern
ends with the Player's finite silence flush. If the splash deadline arrives
before startup audio completes, `Player.Stop` replaces it with finite silence.
The main loop sleeps for 1 ms between iterations: one Player PCM buffer is 48
frames at 48 kHz, also 1 ms. This avoids a full-speed busy loop while keeping
audio refills close to the DMA buffer duration.

Use Arrow Up and Arrow Down to move through the courses. Selection wraps at
both ends. Press Enter to confirm. On Cardputer ADV, the arrow keys use the Fn
layer (`Fn+;` for Up and `Fn+.` for Down). Existing system audio controls remain
available: Fn+= raises volume, Fn+- lowers it, and Fn+M toggles mute.

## Courses and input

The playing screen shows elapsed time and the current question number on the
first 24-dot row, the required ASCII input in 16-dot text, the prompt in 24-dot
text, and a white-on-blue framed input field. The red 24-dot guide at the bottom
continues to scroll while playing. Japanese, English, Chinese, and Korean each
use a guide in the selected course's language. All Languages scrolls the four
complete guides joined with `◇` separators.

- Japanese uses the existing fixed romanizations.
- English words are typed directly.
- Chinese uses lowercase pinyin without tone marks or tone numbers.
- Korean uses a fixed lowercase romanization; Hangul keyboard input is not used.
- All Languages alternates Japanese, English, Chinese, and Korean, with five
  questions from each language.

Type the exact lowercase ASCII input stored with each question. Uppercase key
input is normalized to lowercase, but alternative romanizations, spaces,
punctuation, tone input, and IME composition are not supported. A wrong letter
does not advance the input and increments the miss count. Fn+Backspace produces
DEL and immediately abandons the current game, returning to the menu. Fn+M
toggles mute; Fn+= and Fn+- adjust volume.

After question 20, the result screen freezes and displays total time and misses.
Press Enter to return to the menu. Rankings, persistent scores, networking, and
IME are not implemented.

The title and menu use the embedded Efont 16-dot MGF asset. The scrolling guide
uses Efont 24-dot. The repository tests verify all displayed runes against the
corresponding assets.

The guide starts outside the right edge and loops left at 24 pixels per second,
with a 32-pixel gap between copies. It uses a Viewport-sized RGB565 Surface;
the static title and menu retain the direct rendering path.

## Build and flash

This example imports repository-internal Cardputer ADV drivers and font assets,
so build it from this repository with the TinyGo target used for the board:

```sh
tinygo build \
  -target <cardputer-adv-target> \
  -o rdon-type100.bin \
  ./examples/rdon-type100
```

After reviewing the target and port, it can be flashed separately with the
corresponding command:

```sh
tinygo flash \
  -target <cardputer-adv-target> \
  -port <PORT> \
  ./examples/rdon-type100
```

`<cardputer-adv-target>` may be a locally installed target name or a relative
path to the custom target JSON. Replace `<PORT>` with the actual device port.
Flashing is intentionally not performed as part of this implementation.
