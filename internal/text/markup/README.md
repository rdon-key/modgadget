# text/markup

`<b>...</b>` enables `Style.Bold` while preserving the current font and
colors. Closing the tag restores the complete surrounding Style, including in
nested named styles. Bold is synthetic: glyph ink is extended one pixel to the
right without changing the bitmap font asset or glyph advance.

`markup`は小さなタグ付き文字列を`[]text.Span`へ変換します。対応する構文は
`<style=name>`、`</style>`、`<br>`、`<br/>`、および`<`を表す`<<`です。
style tagは最大16段までnestでき、終了時には直前のStyleへ戻ります。タグ名は
lowercase限定で、style名は小文字英字で始まり、その後に小文字英字、数字、`-`を
使用できます。

`Parser.Styles.Default`はタグ外の文字へ使われます。名前付きStyleは
`StyleSet.Entries`を先頭から完全一致で検索します。`Parse`は結果sliceを確保する
便利APIです。`ParseInto`はcallerが渡したbufferを再利用し、容量不足時はallocation
せずerrorを返します。

```go
styles := text.StyleSet{
    Default: text.Style{
        Font:       text.NewMGFFont(shinonome12.Font),
        Foreground: display.ColorWhite,
        Background: display.ColorBlack,
    },
    Entries: []text.StyleEntry{
        {
            Name: "main",
            Style: text.Style{
                Font:       text.NewMGFFont(efont24.Font),
                Foreground: display.ColorWhite,
                Background: display.ColorBlack,
            },
        },
    },
}

parser := markup.Parser{Styles: styles}
spans, err := parser.Parse(
    "<style=main>今日のニュース</style>",
)
```

inlineのfont、size、color指定、HTML entity、cascade、selector、継承、複数class、
italic、alignment、image、link、CSSには対応しません。
