# MGF1 フォントファイル形式（暫定仕様）

## 1. 目的

MGF（ModGadget Font）は、ModGadgetで使用するビットマップフォントを格納するためのバイナリ形式である。

主な目的は次のとおり。

- 巨大なGoソースを生成せず、フォントを独立したファイルとして扱う
- `go:embed`で埋め込み、実行時にコピーせず参照する
- Unicodeのglyphを検索できる
- 日本語、中国語など、地域によって字形が異なるフォントを選択しやすくする
- 将来、U8g2方式を参考にしたglyph圧縮を利用する
- TinyGoを含む小規模環境で、単純かつallocationを抑えて読み込めるようにする

MGF1では、1ファイルに1フォント、1サブセット、1地域情報を格納する。

## 2. 基本方針

| 項目 | 方針 |
|---|---|
| Magic | 3byteの`MGF` |
| Version | 1byteの数値 |
| Byte order | little-endian |
| FontId | 4byte固定ASCII |
| SubsetId | 4byte固定ASCII |
| Region | 2byteの地域ヒント |
| GlyphCount | `uint16`、最大65,535 |
| 1ファイルの上限 | 最大65,535 glyph |
| Font区・分割番号 | MGF1では持たない |
| LineGap | フォント推奨値として保持 |
| 文字間隔 | ヘッダーには持たない |
| Unicode | glyph index側で扱う |
| 圧縮 | U8g2方式を参考にする予定。詳細は未確定 |

単一regionのフォントが65,535 glyphを超える場合は、MGF1の対象外とする。分割ファイルや「区」の仕様は、必要になった時点で将来versionとして検討する。

## 3. ファイル全体の構成

| 順番 | 領域 | 内容 |
|---:|---|---|
| 1 | Header | ファイル識別、フォント識別、metrics、各領域の位置 |
| 2 | Glyph Index | Unicode code pointからGlyph Recordを検索する索引 |
| 3 | Glyph Data | glyph metricsと圧縮bitmap |
| 4 | 将来拡張 | MGF1では使用しない |

各offsetは、MGFファイル先頭からのbyte位置とする。

Glyph IndexとGlyph Dataの詳細形式は別途決定する。

## 4. MGF1 Header

MGF1のヘッダーは36byte固定とする。

| Offset | Size | 型 | Field | 内容 |
|---:|---:|---|---|---|
| 0 | 3 | byte[3] | Magic | ASCII `MGF` |
| 3 | 1 | uint8 | Version | MGF1は`1` |
| 4 | 4 | byte[4] | FontId | 元フォントの短い識別子 |
| 8 | 4 | byte[4] | SubsetId | サブセットの短い識別子 |
| 12 | 2 | byte[2] | Region | 地域・字形選択のヒント |
| 14 | 2 | uint16 | GlyphCount | 収録glyph数。最大65,535 |
| 16 | 1 | uint8 | Ascent | baselineより上の標準高さ |
| 17 | 1 | uint8 | Descent | baselineより下の標準高さ |
| 18 | 1 | uint8 | LineGap | 推奨する追加行間 |
| 19 | 1 | uint8 | MaxWidth | 全glyph中の最大bitmap幅 |
| 20 | 1 | uint8 | MaxHeight | 全glyph中の最大bitmap高さ |
| 21 | 1 | uint8 | Flags | MGF1では`0` |
| 22 | 2 | uint16 | HeaderSize | MGF1では`36` |
| 24 | 4 | uint32 | IndexOffset | Glyph Indexの開始位置 |
| 28 | 4 | uint32 | GlyphDataOffset | Glyph Dataの開始位置 |
| 32 | 4 | uint32 | FileSize | ファイル全体のbyte数 |

### 4.1 MagicとVersion

先頭4byteで形式とversionを判定する。

- byte 0〜2: `M`、`G`、`F`
- byte 3: version番号

文字列としての`MGF1`ではなく、versionは数値として格納する。

### 4.2 FontId

FontIdは元フォントを識別する4byte固定ASCIIである。

| FontId | 意味 |
|---|---|
| `sh12` | 東雲12 |
| `sp16` | Spleen 8×16 |
| `qn08` | QuanPixel 8×8 |
| `gn16` | GnuFont 16 |

FontIdはUUIDではなく、同じアプリや配布物内で衝突しなければよい。世界的な一意性は要求しない。

MGF1では、FontIdは印字可能なASCII文字4文字を必須とする。終端文字や可変長表現は使用しない。

### 4.3 SubsetId

SubsetIdは、同じ元フォントから生成した異なるサブセットを識別する4byte固定ASCIIである。

| SubsetId | 意味 |
|---|---|
| `full` | 全収録文字版 |
| `ui01` | UI用サブセット |
| `news` | ニュース表示用 |
| `usr1` | ユーザー作成サブセット |
| `jp01` | 日本語向けサブセット第1版 |

SubsetIdも世界的な一意性は要求しない。同じFontId、Regionの組み合わせ内で区別できればよい。

MGF1では、SubsetIdは印字可能なASCII文字4文字を必須とする。

### 4.4 Region

Regionは、字形やfallback順序を決めるための2byteの地域ヒントである。

| Region | 意味 |
|---|---|
| `JP` | 日本向け |
| `CN` | 中国本土向け |
| `TW` | 台湾向け |
| `HK` | 香港向け |
| `KR` | 韓国向け |
| `US` | 米国向け |
| `GB` | 英国向け |
| `00 00` | 指定なし |

Regionは、フォントが収録するUnicode範囲を示すものではない。実際にglyphが存在するかどうかはGlyph Indexで判断する。

fallbackでは、次のような優先順位に利用できる。

1. glyphが存在し、希望Regionと一致するフォント
2. glyphが存在し、Region指定なしのフォント
3. glyphが存在するその他のフォント

Regionの利用方法はrendererまたはアプリ側の方針とし、MGF readerが自動的にfallbackを行う必要はない。

### 4.5 Font identity

MGFファイルの軽量な識別には、次の組み合わせを使用する。

`FontId + SubsetId + Region`

例：

- `sh12 / full / JP`
- `sh12 / ui01 / JP`
- `sh12 / news / JP`
- `sp16 / full / US`

この組み合わせはglyph cacheやデバッグ表示に利用できる。ただし衝突しないことは利用者または生成者の責任とする。

### 4.6 GlyphCount

GlyphCountはlittle-endianの`uint16`である。

- 最大値は65,535
- MGF1では65,536 glyph以上を格納しない
- 複数ファイルへ自動分割する仕様は持たない
- 「Font区」「Shard番号」などのフィールドはMGF1では持たない

### 4.7 Font metrics

基本的なbaseline間隔は次で求める。

`LineAdvance = Ascent + Descent + LineGap`

| Field | 内容 |
|---|---|
| Ascent | baselineより上に確保する標準高さ |
| Descent | baselineより下に確保する標準高さ |
| LineGap | 行と行の間に追加する推奨間隔 |

LineGapはフォントの推奨値であり、アプリやlayout側が別の行間を指定してもよい。

### 4.8 文字間隔

MGF1のヘッダーには文字間隔を持たせない。

文字送りの基本値は各glyphの`AdvanceX`で表現する。追加の文字間隔はlayout側の設定として扱う。

`NextPenX = PenX + Glyph.AdvanceX + Layout.LetterSpacing`

## 5. Flags

MGF1ではFlagsを`0`とする。

将来、次のような属性に使用できる。

- 固定幅フォント
- 固定セル形式
- proportional形式
- icon font
- 圧縮方式
- Glyph Index形式

ただし、意味が確定するまではbitを割り当てない。

## 6. Glyph Index（未確定）

Glyph IndexはUnicode code pointからGlyph Data内のrecord位置を検索するために使用する。

| 案 | 内容 | 特徴 |
|---|---|---|
| 固定長index | `uint32 codepoint + uint32 offset` | 単純でbinary searchしやすい。1 glyphあたり8byte |
| checkpoint index | 一定数ごとにcodepointとoffsetを記録し、block内は差分表現 | 容量を減らせるがdecoderが複雑 |
| codepoint delta | 前glyphとの差分を可変長で格納 | 小さいが直接binary searchしにくい |

MGF1の最初の実装では、東雲12全文字で容量を測定した上で決定する。

UnicodeはGoの`rune`に合わせ、BMPだけに限定しない。

## 7. Glyph Data（未確定）

各Glyph Recordには、少なくとも次の情報が必要になる。

| 項目 | 内容 |
|---|---|
| Width | bitmap幅 |
| Height | bitmap高さ |
| BearingX | pen位置からbitmap左端までの距離 |
| BearingY | baselineからbitmap上端までの距離 |
| AdvanceX | 次のpen位置までの距離 |
| Bitmap length | 圧縮データのbyte数 |
| Bitmap data | 1-bit bitmapまたは圧縮bitmap |

bitmap圧縮にはU8g2方式を参考にした0/1 run-length encodingを検討する。

MGF全体のファイル構造とUnicode索引は独自形式とし、U8g2形式との完全互換は目標にしない。

目標は、bitmap全体をRGB565 scratchへ展開せず、run単位でSurfaceへ描画できること。

## 8. `go:embed`での利用

利用側では、MGFファイルをimmutableな`string`へ埋め込む。

    package assets

    import (
        _ "embed"

        "github.com/rdon-key/modgadget/font"
    )

    //go:embed shinonome12.mgf
    var shinonome12Data string

    var Shinonome12 = font.MustOpen(shinonome12Data)

`font.Open`または`font.MustOpen`は、元のstring全体をコピーせず参照する。

想定API：

    func Open(data string) (Font, error)
    func MustOpen(data string) Font
    func (font *Font) Lookup(r rune) (Glyph, bool)

## 9. Readerの基本検証

MGF readerは最低限、次を検証する。

- Magicが`MGF`
- 対応するVersion
- HeaderSizeが最低必要長以上
- FileSizeが埋め込みデータ長と一致
- IndexOffsetがHeaderSize以降
- GlyphDataOffsetがIndexOffset以降
- 各offsetがFileSize以内
- GlyphCountとIndexの要素数が矛盾しない
- FontIdとSubsetIdが4byteの印字可能ASCII
- Regionが`00 00`または2byteの印字可能ASCII

## 10. 今後決める項目

- Glyph Indexの正式形式
- Glyph Recordの正式byte layout
- U8g2由来RLEの正確なbit形式
- 固定セルフォントのmetrics省略方法
- Flagsのbit割り当て
- CRCまたはchecksumの要否
- 未知version、未知flagsに対するreaderの互換性方針

## 11. MGF1の要約

- MGF1は`go:embed`向けの独立バイナリフォント形式とする。
- ヘッダーは36byte固定。
- Magicは3byteの`MGF`、Versionは1byte。
- FontIdとSubsetIdは、それぞれ4byte固定ASCII。
- Regionは2byteの地域・字形選択ヒント。
- 1ファイルに最大65,535 glyphを格納する。
- 65,535 glyphを超えるフォントの分割仕様は持たない。
- 行送りはAscent、Descent、LineGapで表現する。
- 追加の文字間隔はフォントではなくlayout側で扱う。
- Glyph IndexとGlyph Recordの詳細は、東雲12全文字の実測後に決定する。
- bitmap圧縮はU8g2方式を参考にするが、MGF全体は独自形式とする。
