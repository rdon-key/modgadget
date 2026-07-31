# text/markup

`markup`は、小さなタグ付き文字列を`[]text.Span`へ変換します。

対応タグは`<size=12>`、`<size=16>`、`<size=24>`、`<fg=#RRGGBB>`、
`<bg=#RRGGBB>`、`<br>`、`<br/>`です。styleタグは最大16段までnestでき、
終了タグは開始タグと正しく対応している必要があります。タグ名はlowercase限定で、
タグ内の空白には対応しません。`<<`は文字`<`を表します。

`Parse`は結果sliceを確保する便利APIです。`ParseInto`はcallerが渡したbufferを
再利用し、容量が不足した場合はallocationせずerrorを返します。色は常に
`#RRGGBB`形式です。

```go
parser := markup.Parser{
    Fonts: markup.Fonts{
        Size12: text.MGFFont{Font: shinonome12.Font},
        Size16: text.MGFFont{Font: efont16.Font},
        Size24: text.MGFFont{Font: efont24.Font},
    },
    Foreground: display.ColorWhite,
    Background: display.ColorBlack,
}

var storage [32]text.Span
spans, err := parser.ParseInto(
    storage[:0],
    "通常<size=24><fg=#ff0000>警告</fg></size>",
)
```

HTML entity、任意font名、複数attribute、bold、italic、underline、alignment、
image、link、CSSなどは未対応です。
