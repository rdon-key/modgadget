# text-ticker

Cardputer ADV Viewport scrolling example. It combines static text, a one-shot
scroll from the right, and a multilingual loop using `SetHorizontalScroll`,
`ScrollSpeed`, `ScrollGap`, `ScrollLoop`, and `ScrollFromRight`.

```sh
tinygo build -target=m5stamp-s3a ./examples/text-ticker
```

Flashing is a separate operation and requires the board's target and port.

