# Font and style API

`Font`はrendererが必要とするglyph lookupとmetricsだけを提供する最小interfaceです。
`MGFFont`は、検証済みの`mgf.Font`を`Font`として使うadapterです。

`FontStack`は同じ表示サイズのfontを優先順に検索します。同じcode pointが複数の
fontにある場合は、Primary、Fallbacksの順で最初のglyphを採用します。missing
glyphをreplacement glyphへ自動置換する処理はありません。CJK地域字形の優先順も
applicationがStyleへ設定する`FontStack`で決定できます。language policy自体はFont
APIに含めません。

`Style`はFont、Foreground、Backgroundをすべて持つ完全な見た目の指定です。
部分的なoverrideではありません。bitmap fontの表示サイズはFont自体で決まるため、
Styleに独立したsize fieldはありません。

`StyleSet`はコード内の小さなCSSに近い名前付きStyleの集合ですが、cascade、
selector、継承、複数classはありません。Entriesを先頭から検索し、最初に名前が
一致したStyleを使用します。重複名は避け、style名を一意にすることを推奨します。
タグ外の文字には`StyleSet.Default`を使用します。
