# ModGadget フォント登録手順

この文書では、次の2つの作業手順を説明します。

1. `modgadget-font-assets`へ新しいフォントを登録する
2. `modgadget`へ`modgadget-font-assets`のMGFフォントを取り込む

対象repositoryは、次のように同じ親directoryへ配置されているものとします。

```text
~/work/
├── modgadget/
└── modgadget-font-assets/
```

MGFは、ModGadgetで使用する非圧縮・1bit bitmap font形式です。

- little-endian
- row-major
- MSB-first
- glyph rowはbyte境界へalignment
- runtime lookupはbinary search
- bitmapは元MGF stringへのviewとして扱い、copyしない

---

# 1. `modgadget-font-assets`へフォントを登録する

## 1.1 登録対象フォントの条件

原則として、次を満たすBDFを使用します。

- BDF 2.1または2.2
- Unicode codepointを`ENCODING`へ持つ
- `CHARSET_REGISTRY`が`ISO10646`またはUnicode系
- 再配布可能なlicense
- bitmap形式が1bit
- font全体のAscent、Descent、bounding boxを取得できる

確認例：

```sh
grep -E   '^(FONT |SIZE |FONTBOUNDINGBOX |CHARS |FONT_ASCENT |FONT_DESCENT |CHARSET_REGISTRY |CHARSET_ENCODING )'   path/to/font.bdf
```

代表文字の収録確認例：

```sh
grep -nE '^ENCODING (32|65|92|12354|12450|26085|65509)$' path/to/font.bdf
```

主なcodepoint：

| Codepoint | 文字 |
| ---: | --- |
| 32 | SPACE |
| 65 | A |
| 92 | `\` |
| 12354 | あ |
| 12450 | ア |
| 26085 | 日 |
| 65509 | ￥ |

glyphのadvance幅分布も確認します。

```sh
awk '$1=="DWIDTH"{print $2}' path/to/font.bdf | sort -n | uniq -c
```

---

## 1.2 上流ファイルを取得する

上流archiveやsourceは、正式assetとは分けて作業用directoryへ置きます。

例：

```sh
cd ~/work/modgadget-font-assets

mkdir -p upstream/example-font
curl -L <archive-url> -o upstream/example-font/font.tar.bz2
sha256sum upstream/example-font/font.tar.bz2

mkdir -p upstream/example-font/prebuilt
tar -xjf upstream/example-font/font.tar.bz2   -C upstream/example-font/prebuilt
```

`upstream/`は作業用なので、repositoryへcommitしません。

`.gitignore`例：

```gitignore
# Downloaded upstream sources and generated work files
/upstream/
/work/
```

---

## 1.3 BDFを正式配置する

fontごとにdirectoryを作成します。

例：

```sh
cd ~/work/modgadget-font-assets

mkdir -p fonts/example16
cp upstream/example-font/prebuilt/example16.bdf   fonts/example16/example16.bdf
```

上流ファイルとbyte単位で一致することを確認します。

```sh
cmp   upstream/example-font/prebuilt/example16.bdf   fonts/example16/example16.bdf
```

SHA-256を記録します。

```sh
sha256sum fonts/example16/example16.bdf
```

---

## 1.4 MGFを生成する

`mgfgen`は`modgadget`のGo module内から実行します。

`modgadget-font-assets`直下で、次のように実行してはいけません。

```sh
go run ../modgadget/cmd/mgfgen
```

Goは現在directoryからmoduleを探すため、この形では失敗します。

正しくは、subshell内で`modgadget`へ移動して実行します。

```sh
cd ~/work/modgadget-font-assets

(
    cd ../modgadget
    go run ./cmd/mgfgen         -bdf ../modgadget-font-assets/fonts/example16/example16.bdf         -font-id ex16         -subset-id full         -region JP         -o ../modgadget-font-assets/fonts/example16/example16-full.mgf
)
```

主なoption：

| Option | 内容 |
| --- | --- |
| `-bdf` | 入力BDF |
| `-o` | 出力MGF |
| `-font-id` | 4byte ASCII identifier |
| `-subset-id` | 4byte ASCII subset identifier |
| `-region` | 2byte ASCII region。不要なら省略 |
| `-chars` | subset用UTF-8文字file |
| `-missing` | `error`または`skip` |
| `-line-gap` | MGFのLineGap |
| `-assume-unicode` | charset表記を認識できないBDFをUnicodeとして扱う |

`-font-id`と`-subset-id`は、必ず4文字にします。

例：

```text
FontID:   ef16
SubsetID: full
Region:   JP
```

---

## 1.5 生成結果を確認する

ファイルサイズとSHA-256を確認します。

```sh
wc -c fonts/example16/example16-full.mgf
sha256sum fonts/example16/example16-full.mgf
```

同じ入力とgenerator commitから2回生成し、byte一致を確認します。

```sh
mkdir -p work/example16

(
    cd ../modgadget
    go run ./cmd/mgfgen         -bdf ../modgadget-font-assets/fonts/example16/example16.bdf         -font-id ex16         -subset-id full         -region JP         -o ../modgadget-font-assets/work/example16/example16-full.mgf
)

cmp   fonts/example16/example16-full.mgf   work/example16/example16-full.mgf
```

---

## 1.6 MGF Headerと代表glyphを確認する

一時確認programは`modgadget` repository内へ置きます。

`internal/mgf`はGoの`internal` packageなので、repository外の`/tmp`からはimportできません。

例：

```sh
cd ~/work/modgadget
mkdir -p tmp/checkmgf
```

`tmp/checkmgf/main.go`例：

```go
package main

import (
	"fmt"
	"os"

	"github.com/rdon-key/modgadget/internal/mgf"
)

func main() {
	for _, path := range os.Args[1:] {
		data, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}

		font, err := mgf.Open(string(data))
		if err != nil {
			panic(err)
		}

		header := font.Header()

		fmt.Printf("%s\n", path)
		fmt.Printf("  FontID=%q SubsetID=%q Region=%q\n",
			string(header.FontID[:]),
			string(header.SubsetID[:]),
			string(header.Region[:]))
		fmt.Printf("  GlyphCount=%d Ascent=%d Descent=%d LineGap=%d\n",
			header.GlyphCount,
			header.Ascent,
			header.Descent,
			header.LineGap)
		fmt.Printf("  MaxWidth=%d MaxHeight=%d FileSize=%d\n",
			header.MaxWidth,
			header.MaxHeight,
			header.FileSize)

		for _, r := range []rune{' ', 'A', '\\', 'あ', 'ア', '日', '￥'} {
			glyph, ok := font.Lookup(r)
			fmt.Printf("  U+%04X ok=%v", r, ok)
			if ok {
				fmt.Printf(
					" %dx%d advance=%d bearing=(%d,%d) bitmap=%d",
					glyph.Width,
					glyph.Height,
					glyph.AdvanceX,
					glyph.BearingX,
					glyph.BearingY,
					len(glyph.Bitmap),
				)
			}
			fmt.Println()
		}
	}
}
```

実行例：

```sh
go run ./tmp/checkmgf   ../modgadget-font-assets/fonts/example16/example16-full.mgf

rm -rf tmp/checkmgf
```

確認項目：

- Header field
- GlyphCount
- Ascent、Descent、LineGap
- MaxWidth、MaxHeight
- FileSize
- ASCII glyph
- 日本語glyph
- AdvanceX
- BearingX、BearingY
- Bitmap byte数

---

## 1.7 Licenseとnoticeを登録する

MGFはBDFから生成したbinary assetなので、上流フォントのlicenseやnoticeを引き継ぎます。

license本文やnoticeは次を守ります。

- 原文を変更しない
- 改行を変換しない
- 文字codeを変換しない
- trailing whitespaceを削除しない
- 上流とのbyte一致を維持する

複数サイズが同じ配布物に由来する場合は、noticeをfontごとに重複させず、共有directoryへ置きます。

例：

```sh
NOTICE_DIR=fonts/example-font-1.0-notices
mkdir -p "$NOTICE_DIR"

cp upstream/example-font/prebuilt/COPYRIGHT "$NOTICE_DIR/"
cp upstream/example-font/prebuilt/README* "$NOTICE_DIR/"
```

byte一致確認：

```sh
for file in upstream/example-font/prebuilt/COPYRIGHT             upstream/example-font/prebuilt/README*; do
    cmp "$file" "$NOTICE_DIR/$(basename "$file")" || exit 1
done
```

hash確認：

```sh
sha256sum "$NOTICE_DIR"/*
```

同じ内容のlicenseやnoticeがrepository内にすでにある場合は、重複追加せず既存ファイルを参照します。

---

## 1.8 `.gitattributes`を設定する

Windowsの改行変換や`git diff --check`による上流ファイルの警告を避けます。

例：

```gitattributes
fonts/example16/example16.bdf -text -whitespace
fonts/example16/example16-full.mgf -text
fonts/example-font-1.0-notices/** -text -whitespace
```

意味：

| 属性 | 内容 |
| --- | --- |
| `-text` | Gitによる改行変換を禁止する |
| `-whitespace` | 上流由来のtrailing whitespaceをerror扱いしない |

適用確認：

```sh
git check-attr text whitespace --   fonts/example16/example16.bdf   fonts/example-font-1.0-notices/README
```

期待値：

```text
text: unset
whitespace: unset
```

上流ファイルの空白を削除して`git diff --check`を通してはいけません。  
原文を維持したまま、path単位で検査対象外にします。

---

## 1.9 `MGF-ASSETS.md`へ記録する

最低限、次を記録します。

- 上流release名
- 上流archive SHA-256
- Source BDF path
- Output MGF path
- FontID
- SubsetID
- Region
- GlyphCount
- Ascent
- Descent
- LineGap
- MaxWidth
- MaxHeight
- FileSize
- Source BDF SHA-256
- Output MGF SHA-256
- License／notice path
- 正しい再生成command
- generator repository commit

記載例：

```markdown
## Example Biwidth 16

- Upstream release: `example-font-1.0`
- Upstream release archive SHA-256: `<archive-sha256>`
- Source BDF: `fonts/example16/example16.bdf`
- Output MGF: `fonts/example16/example16-full.mgf`
- FontID: `ex16`
- SubsetID: `full`
- Region: `JP`
- GlyphCount: 12345
- Ascent: 14
- Descent: 2
- LineGap: 0
- MaxWidth: 16
- MaxHeight: 16
- FileSize: 1234567 bytes
- Source BDF SHA-256: `<bdf-sha256>`
- Output MGF SHA-256: `<mgf-sha256>`
- License and upstream notices: see `fonts/example-font-1.0-notices/`.
```

再生成command例：

```sh
(
    cd ../modgadget
    go run ./cmd/mgfgen         -bdf ../modgadget-font-assets/fonts/example16/example16.bdf         -font-id ex16         -subset-id full         -region JP         -o ../modgadget-font-assets/fonts/example16/example16-full.mgf
)
```

---

## 1.10 Gitへ追加する

作業用fileをstageしないように注意します。

stage対象例：

```sh
git add   .gitignore   .gitattributes   MGF-ASSETS.md   fonts/example16   fonts/example-font-1.0-notices
```

確認：

```sh
git diff --cached --check
git diff --cached --stat
git status --short
```

`upstream/`、`work/`、一時fileがstageされていないことを確認します。

commit例：

```sh
git commit -m "add 16px MGF font asset"
git push
```

---

# 2. `modgadget`へasset repositoryのフォントを登録する

## 2.1 作業前確認

両repositoryがcleanであり、asset repositoryがpush済みであることを確認します。

```sh
cd ~/work/modgadget-font-assets

git branch --show-current
git rev-parse HEAD
git status --short
git log -1 --oneline
```

```sh
cd ~/work/modgadget

git branch --show-current
git rev-parse HEAD
git status --short
```

`modgadget`がcleanでない場合は、既存差分を削除せず作業を止めます。

一時directoryが残っている場合は、中身を確認してから削除します。

```sh
find tmp -maxdepth 3 -type f -print
rm -rf tmp
git status --short
```

---

## 2.2 AssetのSHA-256を確認する

コピー前にasset repository側のhashを確認します。

```sh
sha256sum   ~/work/modgadget-font-assets/fonts/example16/example16.bdf   ~/work/modgadget-font-assets/fonts/example16/example16-full.mgf
```

`MGF-ASSETS.md`へ記録された値と一致しない場合は、作業を止めます。

---

## 2.3 Embedded packageを作る

配置例：

```text
internal/fontdata/mgf/example16/
├── data.go
├── data_test.go
└── example16-full.mgf
```

MGFをコピーします。

```sh
cd ~/work/modgadget

mkdir -p internal/fontdata/mgf/example16

cp   ~/work/modgadget-font-assets/fonts/example16/example16-full.mgf   internal/fontdata/mgf/example16/example16-full.mgf
```

byte一致確認：

```sh
cmp   ~/work/modgadget-font-assets/fonts/example16/example16-full.mgf   internal/fontdata/mgf/example16/example16-full.mgf
```

---

## 2.4 `go:embed` packageを実装する

`internal/fontdata/mgf/example16/data.go`例：

```go
// Package example16 provides the embedded Example 16 MGF font.
package example16

import (
	_ "embed"

	"github.com/rdon-key/modgadget/internal/mgf"
)

//go:embed example16-full.mgf
var data string

// Font is the embedded full Example 16 font.
var Font mgf.Font = mgf.MustOpen(data)
```

原則としてexportするものは次だけです。

```go
var Font mgf.Font
```

守ること：

- raw `data`は非公開
- `go:embed`変数は1つだけ
- `[]byte(data)`をpackage変数へ保持しない
- MGF全体をcopyしない
- bitmapをcopyしない
- `unsafe`を使わない
- init時にgoroutineを起動しない
- map cacheを作らない

`mgf.MustOpen`はpackage初期化時にMGF全体を検証します。

---

## 2.5 Embedded asset testを追加する

`data_test.go`では、embedded stringそのものを検査します。

確認項目：

- `len(data)`
- SHA-256
- Header
- 代表glyph
- ASCIIと日本語の幅
- allocation 0

SHA-256確認例：

```go
if len(data) != expectedSize {
	t.Fatalf("data size = %d", len(data))
}

hash := fmt.Sprintf("%x", sha256.Sum256([]byte(data)))
if hash != expectedSHA256 {
	t.Fatalf("SHA-256 = %s", hash)
}
```

Header確認項目：

```text
FontID
SubsetID
Region
GlyphCount
Ascent
Descent
LineGap
MaxWidth
MaxHeight
FileSize
```

代表glyph例：

```go
[]rune{' ', 'A', '\', 'あ', 'ア', '日', '￥'}
```

allocation確認例：

```go
allocations := testing.AllocsPerRun(100, func() {
	_, _ = Font.Lookup('あ')
})
if allocations != 0 {
	t.Fatalf("allocations = %v", allocations)
}
```

最低限、次を測定します。

- lookup hit
- lookup miss
- Header
- GlyphCount
- LineHeight

---

## 2.6 Rendererとの互換性をtestする

既存の`text.MGFFont`を使用します。

```go
font := text.MGFFont{
	Font: example16.Font,
}
```

新しいadapterはfontごとに作りません。

確認項目：

- FontMetrics
- LineHeight
- baseline
- AdvanceX
- 複数サイズ混在
- row scratch
- allocation 0

複数サイズを同じlineへ置いた場合、line metricsは各fieldの最大値になります。

```text
Ascent  = max(all spans)
Descent = max(all spans)
LineGap = max(all spans)
```

bitmap上端：

```text
bitmapY = baselineY - BearingY
```

例：

| Font | BearingY | baseline 30のbitmap top |
| --- | ---: | ---: |
| 12px | 10 | 20 |
| 16px | 14 | 16 |
| 24px | 22 | 8 |

pen advanceは各glyphの`AdvanceX`の合計です。

例：

```text
12px「あ」 = 12
16px「あ」 = 16
24px「あ」 = 24
合計       = 52
```

---

## 2.7 Row scratchを確認する

MGF rendererは1 scanline分のRGB565をcaller-provided scratchへ展開します。

必要byte数：

```text
MaxWidth * 2
```

例：

| 最大幅 | 必要scratch |
| ---: | ---: |
| 12px | 24 bytes |
| 16px | 32 bytes |
| 24px | 48 bytes |

24pxの場合：

```go
var scratch [24 * 2]byte
```

不足時は`BeginRect`前にerrorになります。

例：

```text
have 47 bytes, need 48
```

既存exampleが12pxしか使用していない場合は、そのexampleのscratchを48byteへ増やす必要はありません。  
24px fontを実際にimportして描画するexampleやtag rendererだけ48byteを用意します。

---

## 2.8 Licenseとnoticeをコピーする

asset repositoryから上流noticeをコピーします。

```sh
cd ~/work/modgadget

mkdir -p LICENSES/example-font-1.0

cp   ~/work/modgadget-font-assets/fonts/example-font-1.0-notices/*   LICENSES/example-font-1.0/
```

上流とbyte一致することを確認します。

```sh
for file in ~/work/modgadget-font-assets/fonts/example-font-1.0-notices/*; do
    cmp "$file" "LICENSES/example-font-1.0/$(basename "$file")" || exit 1
done
```

同一内容のlicenseやnoticeが既に`LICENSES/`へ存在する場合は重複追加しません。

その場合は、provenance文書で既存ファイルへの対応を記録します。

---

## 2.9 `.gitattributes`を追加する

例：

```gitattributes
internal/fontdata/mgf/example16/example16-full.mgf -text
LICENSES/example-font-1.0/** -text -whitespace
```

目的：

- MGFの改行変換防止
- noticeのbyte一致維持
- 上流由来trailing whitespaceを保持
- `git diff --check`を正常に通す

適用確認：

```sh
git check-attr text whitespace --   internal/fontdata/mgf/example16/example16-full.mgf   LICENSES/example-font-1.0/README
```

---

## 2.10 Provenance READMEを更新する

更新先：

```text
internal/fontdata/mgf/README.md
```

記録項目：

- source repository名
- source repository commit
- generator repository commit
- Source BDF path
- Source MGF path
- Embedded output path
- BDF SHA-256
- MGF SHA-256
- FileSize
- license／notice path
- binaryを手編集しないこと
- asset repositoryの`MGF-ASSETS.md`を参照すること

例：

```markdown
| Font | Source BDF | Source MGF | Embedded output | BDF SHA-256 | MGF SHA-256 | FileSize |
| --- | --- | --- | --- | --- | --- | ---: |
| Example 16 | `fonts/example16/example16.bdf` | `fonts/example16/example16-full.mgf` | `internal/fontdata/mgf/example16/example16-full.mgf` | `<bdf-sha256>` | `<mgf-sha256>` | 1234567 |
```

---

## 2.11 Testを実行する

font package：

```sh
go test ./internal/fontdata/mgf/example16
```

MGF runtimeとrendererも回帰確認します。

```sh
go test ./internal/mgf
go test ./internal/text
```

race test：

```sh
go test -race   ./internal/mgf   ./internal/fontdata/mgf/example16   ./internal/text
```

vet：

```sh
go vet   ./internal/mgf   ./internal/fontdata/mgf/example16   ./internal/text
```

差分確認：

```sh
git diff --check
```

通常Go環境では、TinyGo専用の`machine` packageを使うpackageが失敗する場合があります。  
今回変更したmachine非依存packageの結果と分けて扱います。

---

## 2.12 Assetのlinkを確認する

`go:embed` assetは、そのpackageがimportされた場合だけbinaryへlinkされます。

確認方法：

1. 新しいfont packageをimportしないhost binaryを作る
2. 新しいfont packageをimportし、`Font.GlyphCount()`を参照するhost binaryを作る
3. binary sizeを比較する
4. 一時programとbinaryを削除する

増加量は、おおむねMGF assetの合計サイズ＋package初期化codeです。

assetを登録しただけで、未importの既存exampleが巨大化してはいけません。

---

## 2.13 最終確認とcommit

byte一致：

```sh
cmp   ~/work/modgadget-font-assets/fonts/example16/example16-full.mgf   ~/work/modgadget/internal/fontdata/mgf/example16/example16-full.mgf
```

SHA-256：

```sh
sha256sum   internal/fontdata/mgf/example16/example16-full.mgf
```

Git確認：

```sh
git status --short
git diff --stat
git diff --check
```

stage例：

```sh
git add   .gitattributes   LICENSES/example-font-1.0   internal/fontdata/mgf/README.md   internal/fontdata/mgf/example16   internal/text/example16_test.go
```

stage後：

```sh
git diff --cached --check
git status --short
```

commit例：

```sh
git commit -m "add 16px embedded font"
git push
```

---

# 3. 現在登録済みの例

## Efont Biwidth 16

```text
Asset repository commit:
66b23862fe6ec38ef362ef74596912231ec14a51

Generator commit:
36141f000687df39c3dad106f5455f55be64e6b0

FontID:     ef16
SubsetID:   full
Region:     JP
GlyphCount: 24618
Ascent:     14
Descent:    2
LineGap:    0
MaxWidth:   16
MaxHeight:  16
FileSize:   1167336
```

```text
BDF SHA-256:
2dd69898adba95a4bb47a7490b54ccb0fc95bd59007fc63fe2c6bb29a9bc5cb5

MGF SHA-256:
0cbbcc0b0a3845be11d5cd958c2ea092afa6fdd82be9ae82f6d1a87274e9ea16
```

## Efont Biwidth 24

```text
Asset repository commit:
66b23862fe6ec38ef362ef74596912231ec14a51

Generator commit:
36141f000687df39c3dad106f5455f55be64e6b0

FontID:     ef24
SubsetID:   full
Region:     JP
GlyphCount: 30641
Ascent:     22
Descent:    2
LineGap:    0
MaxWidth:   24
MaxHeight:  24
FileSize:   2706726
```

```text
BDF SHA-256:
f03ad7d046b2b7e976bfba89f500117ef8d11c370055ba4adede866023754ad6

MGF SHA-256:
d87645e7b45cbf9e9758349a9a337bd38ef832d781e96c19d0596f113ca8f4a7
```

---

# 4. チェックリスト

## `modgadget-font-assets`

- [ ] 上流archiveのSHA-256を記録した
- [ ] BDFのUnicode charsetを確認した
- [ ] 代表glyphの収録を確認した
- [ ] BDFを`fonts/`へ正式配置した
- [ ] 上流BDFとの`cmp`が成功した
- [ ] `modgadget` module内からMGFを生成した
- [ ] MGFを再生成してbyte一致を確認した
- [ ] MGF Headerと代表glyphを確認した
- [ ] BDF／MGF SHA-256を記録した
- [ ] license／noticeを原文のまま保存した
- [ ] `.gitattributes`でbyte一致を保護した
- [ ] `MGF-ASSETS.md`を更新した
- [ ] `upstream/`と`work/`をstageしていない
- [ ] `git diff --cached --check`が成功した

## `modgadget`

- [ ] asset repositoryのcommitとclean状態を確認した
- [ ] コピー前にSHA-256を確認した
- [ ] MGFを`internal/fontdata/mgf/<font>/`へコピーした
- [ ] コピー元との`cmp`が成功した
- [ ] `go:embed`変数が1つだけである
- [ ] raw dataをexportしていない
- [ ] `mgf.MustOpen`で初期化している
- [ ] embedded dataのsizeとSHA-256をtestした
- [ ] Headerと代表glyphをtestした
- [ ] lookupが0 allocationである
- [ ] rendererでmetricsとbaselineをtestした
- [ ] 最大幅に必要なrow scratchをtestした
- [ ] license／noticeをbyte一致で保存した
- [ ] `.gitattributes`を追加した
- [ ] provenance READMEを更新した
- [ ] 未import時にassetがlinkされないことを確認した
- [ ] race testとvetが成功した
- [ ] `git diff --check`が成功した
