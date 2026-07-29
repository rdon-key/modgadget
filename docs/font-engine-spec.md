# ModGadget Font Engine Specification

Status: Draft
Version: 0.1

## 1. 概要

ModGadgetは、小型ディスプレイ向けのGUIエンジンである。

Font EngineはModGadgetの基盤コンポーネントとして、Unicodeテキストをグリフ列へ変換し、可変幅、baseline、複数フォント、fallback、span単位の装飾を扱う。

Font Engineは単独の文字描画APIではなく、将来の次の機能を支える。

* Markdown文書表示
* タグ付きテキスト
* 複数フォントの混在
* CJK全文字集合
* 欧文可変幅フォント
* 色、背景色、太字、反転、点滅
* インライン画像
* アニメーション
* z-indexによる重ね合わせ
* スクロール可能な文書表示
* framebufferを使用しない矩形streaming描画

Font Engine自身は、Markdown解析、画像描画、z-index、animationを直接担当しない。

Font Engineは、DocumentおよびSpanからグリフ配置情報を生成し、RendererおよびGUI Sceneへ渡す。

## 2. 設計目標

### 2.1 Unicodeネイティブ

Font Engine内部ではUTF-8文字列をUnicode code pointとして扱う。

Shift-JISやFONTX固有の文字コードを内部表現には使用しない。

最低限、次の文字集合を扱える設計とする。

* ASCII
* Latin
* ひらがな
* カタカナ
* CJK記号
* CJK統合漢字
* CJK拡張領域
* 簡体字中国語
* 繁体字中国語
* 日本語
* 韓国語
* Unicode補助平面

Unicode code pointは32bitで保持する。

### 2.2 CJKと欧文の両立

CJK固定幅フォントだけを前提としない。

次の両方を同じモデルで扱う。

* 12×12などの固定幅CJK bitmap font
* 可変幅の欧文bitmap font

グリフ幅と文字送り幅は別の値として保持する。

### 2.3 baselineベースの配置

文字配置はセル左上基準ではなく、baselineを基準とする。

これにより、次を正しく扱う。

* `g`、`p`、`y`などbaselineより下へ出る文字
* アクセント記号
* 異なるサイズのフォント
* CJKと欧文の混在
* インライン画像
* baseline shift

### 2.4 framebuffer非依存

Font Engineはframebufferの存在を前提としない。

Rendererはグリフや背景をRGB565などの矩形データへ展開し、Display Backendへstreaming転送できる。

### 2.5 データ形式と描画処理の分離

Font Engineはフォントデータの保存方法を意識しない。

フォントは次の形式から提供可能とする。

* Go sourceへ埋め込まれたデータ
* ModGadget独自binary font
* 内蔵Flash
* 外部Flash
* SDカード
* ネットワークから取得したfont pack

Text Layoutは共通のFace interfaceのみを使用する。

## 3. 非目標

初期バージョンでは次を必須としない。

* OpenType完全対応
* TrueTypeの実行時rasterize
* 複雑なshaping engine
* bidirectional text完全対応
* Unicode line breaking完全対応
* 高度な禁則処理
* ligature完全対応
* アンチエイリアスフォント

ただし、将来これらを追加できない構造にはしない。

## 4. コンポーネント構成

Font Engine周辺は次の層で構成する。

```
Markdown / Markup
        ↓
Document / Inline Nodes
        ↓
Span Style Resolution
        ↓
Font Resolution
        ↓
Text Shaping
        ↓
Line Layout
        ↓
Glyph Runs
        ↓
Renderer
        ↓
Display Backend
```

### 4.1 Font Loader

フォントデータを読み込み、Faceとして提供する。

### 4.2 Font Manager

font family、weight、slant、languageなどから使用するFaceを解決する。

### 4.3 Text Shaper

Unicode文字列をglyph ID列へ変換する。

初期実装では1 runeを1 glyphとして扱ってよい。

将来は次を扱う。

* Variation Selector
* combining mark
* grapheme cluster
* ligature
* script固有のshaping

### 4.4 Line Layout

glyph advance、bearing、kerning、line metricsを使用して、行内の位置を決定する。

### 4.5 Renderer

配置済みglyphをbitmapから表示形式へ展開する。

Rendererは次を担当する。

* 前景色
* 背景色
* 反転
* synthetic bold
* underline
* strikethrough
* blink phase
* clipping
* dirty rectangleへの描画

## 5. Font Metrics

フォント全体のmetricsは次を持つ。

```
type FontMetrics struct {
    Ascent             int16
    Descent            int16
    LineGap            int16
    UnderlinePosition  int16
    UnderlineThickness int16
}
```

### 5.1 Ascent

baselineから上方向へ必要な高さ。

正の値で保持する。

### 5.2 Descent

baselineから下方向へ必要な高さ。

正の値で保持する。

### 5.3 LineGap

隣接する行との追加間隔。

### 5.4 Line Height

基本的な行の高さは次で求める。

```
LineHeight = Ascent + Descent + LineGap
```

複数フォントが一行に混在する場合は、使用するFaceのmetricsから必要な行領域を決定する。

## 6. Glyph Metrics

各グリフは次の情報を持つ。

```
type Glyph struct {
    BitmapOffset uint32

    Width    int16
    Height   int16

    AdvanceX int16
    BearingX int16
    BearingY int16
}
```

### 6.1 Width / Height

グリフbitmapの実際の幅と高さ。

### 6.2 AdvanceX

次の文字へ進む距離。

`Width`と同じとは限らない。

空白文字はbitmapを持たず、`AdvanceX`のみを持つことができる。

### 6.3 BearingX

現在のpen位置からグリフbitmap左端までの距離。

描画X座標は次で求める。

```
GlyphX = PenX + BearingX
```

### 6.4 BearingY

baselineからグリフbitmap上端までの距離。

画面Y座標が下方向へ増加する場合、描画Y座標は次で求める。

```
GlyphY = BaselineY - BearingY
```

### 6.5 Pen Advance

グリフ配置後、penを次へ進める。

```
PenX += AdvanceX
```

## 7. Face API

フォントの一つの書体とサイズをFaceと呼ぶ。

```
type Face interface {
    Metrics() FontMetrics

    LookupRune(r rune) (GlyphID, bool)
    Glyph(id GlyphID) Glyph
    Bitmap(id GlyphID) Bitmap
}
```

`GlyphID`はFace内部のglyphを識別する値である。

```
type GlyphID uint32
```

`Bitmap`は保存形式に依存しない読み取り専用のbitmap参照を表す。

初期実装ではimmutable stringまたはbyte sliceを利用できる。

## 8. Optional Font Interfaces

すべてのFaceに高度な機能を要求しない。

必要な機能はoptional interfaceとして分離する。

### 8.1 Kerning

```
type KerningFace interface {
    Kerning(left, right GlyphID) int16
}
```

### 8.2 Variation Sequence

```
type VariationFace interface {
    LookupVariant(base, variation rune) (GlyphID, bool)
}
```

### 8.3 Glyph Cluster

```
type ClusterFace interface {
    LookupCluster(cluster TextCluster) ([]GlyphID, bool)
}
```

## 9. Font Style

Spanから指定されるフォント関連属性は次のように表す。

```
type FontStyle struct {
    Family        string
    Weight        FontWeight
    Slant         FontSlant
    PixelSize     int16
    LetterSpacing int16
    BaselineShift int16
    Language      string
}
```

### 9.1 Family

使用するfont family名。

例:

* `spleen`
* `shinonome`
* `noto-sans`
* `noto-cjk-jp`
* `noto-cjk-sc`
* `noto-cjk-tc`
* `noto-cjk-kr`

### 9.2 Weight

```
type FontWeight uint16
```

代表値:

* 400: Regular
* 500: Medium
* 600: SemiBold
* 700: Bold

### 9.3 Slant

```
type FontSlant uint8

const {
    SlantNormal FontSlant = iota
    SlantItalic
    SlantOblique
}
```

### 9.4 PixelSize

希望する表示サイズ。

bitmap fontでは、利用可能なFaceの中から最も近いサイズを選択する。

### 9.5 Language

CJK統合漢字の地域別字形選択に使用する。

例:

* `ja`
* `zh-CN`
* `zh-TW`
* `ko`

## 10. Paint Style

描画に関する属性はFont Styleと分離する。

```
type PaintStyle struct {
    Foreground Color565
    Background Color565

    Invert        bool
    Blink         bool
    Underline     bool
    Strikethrough bool

    SyntheticBold bool
}
```

### 10.1 Invert

前景色と背景色を交換する。

### 10.2 Blink

Rendererへblink対象であることを伝える。

Font Engine自身はtickerやgoroutineを開始しない。

blink phaseはGUI EngineまたはAnimation Engineから渡される。

### 10.3 Bold

Bold faceが存在する場合は、そのFaceを優先する。

Bold faceが存在しない場合のみsynthetic boldを使用できる。

## 11. Span

テキストの一部分へ適用する属性単位をSpanと呼ぶ。

```
type TextSpan struct {
    Text  string
    Font  FontStyle
    Paint PaintStyle

    ID    string
    Class string
}
```

SpanはHTMLの`span`に近い役割を持つ。

Spanは次を変更できる。

* font family
* font size
* weight
* slant
* language
* foreground
* background
* bold
* invert
* blink
* underline
* strikethrough
* letter spacing
* baseline shift

Spanはz-indexや絶対座標を直接管理しない。

z-indexはGUI Scene Nodeの責務とする。

## 12. Markup例

Markdownから生成される拡張タグは次のような形式を想定する。

```
通常の文章

[font=shinonome]日本語[/font]

[font=spleen][color=yellow][b]Hello[/b][/color][/font]

[lang=zh-CN][font=noto-cjk-sc]简体中文[/font][/lang]

[invert]警告[/invert]

[blink][color=red]更新があります[/color][/blink]
```

タグ構文はFont Engineの仕様には含めない。

Markup Parserはタグを解析して`TextSpan`列へ変換する。

## 13. Font Resolution

Font ManagerはFont StyleからFaceを選択する。

```
type FontManager interface {
    Resolve(style FontStyle) FontSet
}
```

FontSetはprimary Faceとfallback chainを持つ。

```
type FontSet struct {
    Primary   Face
    Fallbacks []Face
    Missing   Face
}
```

解決順の例:

1. 指定familyの指定weightおよびslant
2. 指定familyのRegular
3. 言語別CJK Face
4. Symbol Face
5. Missing Glyph Face

`[font=...]`タグはprimary Faceを変更する。

指定Faceにglyphがない場合でもfallback chainは利用できる。

## 14. CJK Font Faces

CJKの地域別字形を扱うため、同じUnicode code pointに対して複数のFaceを提供できる。

例:

* Japanese Face
* Simplified Chinese Face
* Traditional Chinese Face
* Korean Face

Font ManagerはSpanまたはDocumentのLanguage属性を使用してFaceを選択する。

一つの巨大な共通Faceだけに依存しない。

## 15. Text Cluster

将来の複数code point対応のため、内部表現としてText Clusterを定義する。

```
type TextCluster struct {
    Runes []rune
}
```

初期実装では各runeを一つのclusterとして扱う。

将来は次を一つのclusterとして扱う。

* 基底文字とVariation Selector
* 基底文字と結合文字
* grapheme cluster
* ligature対象文字列

## 16. Glyph Run

同じFaceとPaint Styleを共有する連続したグリフをGlyph Runと呼ぶ。

```
type GlyphPlacement struct {
    ID GlyphID

    X int16
    Y int16
}

type GlyphRun struct {
    Face   Face
    Paint  PaintStyle
    Glyphs []GlyphPlacement
}
```

Glyph RunはRendererへ渡す最小単位となる。

異なるfont family、weight、色、背景色、blink属性を持つ場合は別のRunへ分割する。

## 17. Line Layout

行はbaselineと複数のGlyph Runを持つ。

```
type Line struct {
    BaselineY int16
    Ascent    int16
    Descent   int16

    Runs []GlyphRun
}
```

Line Layoutは次を担当する。

* 可変幅glyph配置
* kerning
* letter spacing
* Span境界
* font fallback
* baseline統一
* baseline shift
* 改行
* 画面幅による折り返し

## 18. インライン画像との関係

将来、テキストと画像を同じ行へ配置する。

```
type InlineNode interface {
    InlineMetrics() InlineMetrics
}

type InlineText struct {
    Spans []TextSpan
}

type InlineImage struct {
    Source ImageRef
    Width  int16
    Height int16
    Align  InlineAlign
}
```

Inline Imageは次の配置方法を持てる。

* baseline
* top
* middle
* bottom

画像のdecodeやanimationはFont Engineの責務ではない。

Text Layoutと共通のInline Layout層が文字と画像を配置する。

## 19. GUI Engineとの境界

Font EngineおよびText Layoutの出力は、GUI EngineのScene Nodeへ変換される。

Scene Nodeは次を持つ。

* position
* bounds
* z-index
* visibility
* clipping
* animation
* dirty state

Spanはインラインスタイルを表す。

Scene Nodeは重ね合わせと動作を表す。

両者を混同しない。

## 20. Rendering

RendererはGlyph RunをDisplay Backendへ描画する。

描画手順:

1. glyph bitmapを取得する
2. Paint Styleを適用する
3. 必要な矩形をRGB565へ展開する
4. Display Backendへ連続転送する

Rendererは1ピクセル単位の`SetPixel`を基本描画方式にしない。

矩形単位のstreamingを使用する。

## 21. Font Data Format

実機用フォント形式はUnicodeネイティブなbinary formatとする。

最低限、次を保持する。

* format version
* font metrics
* glyph count
* Unicode mapping
* glyph records
* bitmap blob
* optional kerning table
* optional variation table
* optional metadata

コードポイントは32bitで保持する。

可変幅とbaselineを扱うため、glyphごとにmetricsを持てる形式とする。

FONTX2はimport対象または参考仕様とするが、内部形式としては使用しない。

主な理由:

* Shift-JIS依存
* 16bit文字コード
* CJK拡張領域を扱えない
* 可変幅およびbaseline情報が不足する

## 22. Font Conversion

`modgadget-fonts`は次の入力を段階的に扱う。

* BDF
* FONTX2
* 将来的にTTFまたはOTF

出力形式:

* Go source
* ModGadget binary font
* subset font
* full font pack

変換時に次の情報を保持する。

* Unicode mapping
* width
* height
* advance
* bearing
* ascent
* descent
* line gap
* bitmap
* font family
* weight
* slant
* language

## 23. 初期実装範囲

最初のFont Engineでは次を実装する。

1. BDFからfont metricsを取得する
2. glyphごとのwidth、heightを保持する
3. advance、bearing、baselineを保持する
4. 可変幅glyphを配置する
5. 複数Faceを登録する
6. Span単位でFaceを切り替える
7. glyph単位のfallbackを行う
8. foregroundとbackgroundを描画する
9. invertを描画する
10. synthetic boldを描画する
11. 複数行の折り返しを行う

初期実装では次を後回しにできる。

* kerning
* Variation Selector
* combining mark
* ligature
* complex shaping
* italic合成
* animation
* z-index
* inline image

ただし公開型とデータ形式は、これらの追加を妨げないものとする。

## 24. 初期API例

```
spans := []text.TextSpan{
    {
        Text: "日本語 ",
        Font: text.FontStyle{
            Family:  "shinonome",
            Language: "ja",
        },
        Paint: text.PaintStyle{
            Foreground: display.ColorWhite,
            Background: display.ColorBlack,
        },
    },
    {
        Text: "Hello",
        Font: text.FontStyle{
            Family: "spleen",
            Weight: 700,
        },
        Paint: text.PaintStyle{
            Foreground: display.ColorYellow,
            Background: display.ColorBlack,
        },
    },
}

lines, err := layout.LayoutSpans(
    spans,
    layout.Options{
        Width: 240,
    },
)
if err != nil {
    return err
}

return renderer.DrawLines(lines)
```

## 25. 設計原則

* Font EngineはGUI Engineの基盤である
* Font EngineとMarkup Parserを分離する
* Font StyleとPaint Styleを分離する
* Text LayoutとRendererを分離する
* Font DataとFont Storageを分離する
* CJKを特別扱いするのではなく、一般的なglyph model上で扱う
* 固定幅フォントを前提にしない
* baselineを必須概念とする
* Unicode code pointを32bitで扱う
* fallbackを標準機能とする
* Span単位でフォントを変更できる
* framebufferを必須にしない
* 一文字ごとのSetPixelを基本方式にしない
* 将来の画像、animation、z-indexを妨げない
* 初期実装は小さく保ちつつ、データ構造は拡張可能にする

## 26. 最終目標

ModGadget Font Engineは、単なる文字描画ライブラリではない。

Unicode CJK全文字集合と欧文可変幅フォントを同じbaselineモデルで扱い、Span単位のフォント切り替えと装飾を適用し、小型ディスプレイ向けGUI Engineへ効率的なglyph layoutを提供する。

Font Engineは、Markdown文書、画像、animation、z-indexを持つModGadget GUI Engineのテキスト基盤となる。

