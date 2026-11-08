# テンプレートの作成方法

## テンプレート機能

### 利用可能なデータ構造
```go
type TemplateData struct {
    Items []QuizItem  // クイズデータのスライス
}

type QuizItem struct {
    Question string    // 問題文
    Answer   string    // 答え
    Spell    string    // 原語表記（英語表記）
    Tags     []string  // タグ
    Comments []string  // コメント（補足説明など）
    Criteria *Criteria // 判定基準（判定基準が無い場合はnil）
}

type Criteria struct {
    OK     []string // 別解（criteria.ok）
    NG     []string // 誤答（criteria.ng）
    Repeat []string // もう一度（criteria.repeat）
}
```

`Criteria` は，YAMLに `criteria` が無い場合は `nil` になります．
判定基準の有無で出力を切り替えるときは `{{if .HasCriteria}}...{{end}}` を使ってください．
`HasCriteria` は，`criteria` が無い場合に加えて，`criteria: {}` や `ok: []` のように中身が空の場合も偽になります．
（`{{if .Criteria}}` は `criteria` が書かれているかどうかだけを判定するため，中身が空でも真になります．）

### 判定基準の個別アクセス

`QuizItem` では，判定基準の各項目に以下の変数でアクセスできます（いずれも `[]string`．該当キーがない場合や `Criteria` が `nil` の場合は空）．

| 変数 | 内容 | 対応するキー |
|------|------|--------------|
| `{{.OK}}` | 別解（正解） | `criteria.ok` |
| `{{.NG}}` | 誤答 | `criteria.ng` |
| `{{.Repeat}}` | もう一度 | `criteria.repeat` |

`{{.Criteria.OK}}` / `{{.Criteria.NG}}` / `{{.Criteria.Repeat}}` でも同じ値を参照できますが，
判定基準が無い問題では `Criteria` が `nil` のためエラーになります．`{{if .HasCriteria}}` で囲むか，`{{.OK}}` などを使ってください．

```text
{{if .OK}}別解: {{quoteAll .OK ""}}（{{len .OK}}件）{{end}}
{{if .NG}}誤答: {{quoteAll .NG "は誤答"}}{{end}}
{{range .Repeat}}- {{addQuotes .}}はもう一度
{{end}}
```

出力例（`ok: [正答1, 正答2]`, `ng: [誤答1]`, `repeat: [もう一度1]` の場合）:

```text
別解: 「正答1」「正答2」（2件）
誤答: 「誤答1」は誤答
- 「もう一度1」はもう一度
```

### 利用可能なテンプレート関数

| 関数名 | 説明 | 使用例 |
|--------|------|--------|
| `formatCriteria` | criteriaを専用形式でフォーマット | `{{formatCriteria .Criteria}}` |
| `quoteAll` | 各要素に「」を付けて**区切り文字なしで**連結し，第2引数の文字列を末尾に付ける（第2引数は区切り文字ではない．リストが空なら空文字） | `{{quoteAll .NG "は誤答"}}` → `「誤答1」「誤答2」は誤答` |
| `csvRow` | 引数を1行分のCSVレコードに整形（カンマ・`"`・改行を含む値は自動でクォート．末尾の改行は含まない） | `{{csvRow .Question .Answer}}` |
| `csvField` | 1つの値をCSVのフィールドとしてエスケープ | `{{csvField .Question}}` |
| `csvRowSafe` | `csvRow` と同じだが，`=` / `+` / `-` / `@` / タブ / CR で始まる値（先頭の半角スペースは無視）の先頭に `'` を付け，表計算ソフトで数式として解釈されないようにする | `{{csvRowSafe .Question .Answer}}` |
| `csvFieldSafe` | `csvField` と同じだが，`csvRowSafe` と同じ規則で値の先頭に `'` を付ける | `{{csvFieldSafe .Question}}` |
| `addQuotes` | 「」引用符を追加 | `{{addQuotes .Answer}}` |
| `join` | 文字列スライスを結合 | `{{join .Strings ","}}` |
| `upper` | 大文字に変換 | `{{upper .Question}}` |
| `lower` | 小文字に変換 | `{{lower .Answer}}` |
| `replace` | 文字列置換 | `{{replace .Text "old" "new"}}` |
| `add` | 数値の加算 | `{{add $index 1}}` |
| `len` | スライスの長さ（Go標準） | `{{len .Items}}`, `{{len .OK}}` |
| `now` | 現在日時 | `{{now}}` |
| `escapeHTML` | HTMLの特殊文字（`<` `>` `&` `"` `'`）をエスケープ．Markdownなど自動エスケープされないテンプレートで使用（HTMLテンプレートでも二重エスケープされない） | `{{escapeHTML .Question}}` |
| `nl2br` | 改行（`\n`・`\r\n`・`\r`）を `<br>` に置き換えて値を1行にまとめる（Markdownテンプレート用．エスケープは行わないため `escapeHTML` のあとに使う） | `{{escapeHTML .Question \| nl2br}}` |
| `safeHTML` | 値をエスケープせずにHTMLとして出力（HTMLテンプレート用．信頼できる値にのみ使用） | `{{join .Comments "<br>" \| safeHTML}}` |

#### 注意
- `add`は数値の加算に使います．デフォルトでは`$index`は0から始まるため、1を加えることで1から始まる番号付けが可能です。
- `formatCriteria`は判定基準を以下のようにフォーマットします:

```text
「別解1」「別解2」／「誤答1」「誤答2」は誤答／「もう一度1」「もう一度2」はもう一度
```
  - 別解は`ok`，誤答は`ng`，もう一度は`repeat`キーの値を使用します．
  - 空のリストは出力に含めません．判定基準が無い（`.Criteria` が `nil`）場合は空文字を返します．
- `nl2br`は，Markdownテンプレートで改行を含む値（YAMLの `|` で書いた複数行の問題文など）を出力するときに使います．
  - 改行をそのまま出力すると，`**Q:** ...` の段落が空行で途切れたり，2行目以降の行頭の `#`・`-`・`>` などが見出し・リスト・引用として解釈されたりします．`nl2br`は改行を `<br>` に置き換えて値を1行にまとめるため，こうした崩れが起きません．
  - `{{escapeHTML .Question | nl2br}}` のように，`escapeHTML` のあとに使ってください（先に `nl2br` を使うと，`<br>` もエスケープされて `&lt;br&gt;` になります）．
  - 組み込みの `quiz_template.md` は，問題文・答え・コメント・判定基準をこの形で出力しています．
  - HTMLテンプレートでは結果がエスケープされ `&lt;br&gt;` と出力されるため，使わないでください．

```text
{{escapeHTML .Question | nl2br}}
```

出力例（問題文が `1行目` と `- 2行目` の2行の場合）:

```text
1行目<br>- 2行目
```

### 以前のバージョンからの移行

`Criteria` は以前は `map[string][]string` でしたが，専用の構造体に変わりました．
`{{formatCriteria .Criteria}}` はそのまま使えます．
`{{if .Criteria}}` もそのまま使えますが，`criteria: {}` のように中身が空の場合に真になるため，`{{if .HasCriteria}}` への書き換えをおすすめします．
マップのキーで参照していた箇所は，以下のように書き換えてください．

| 以前の書き方 | 現在の書き方 |
|--------------|--------------|
| `{{.Criteria.ok}}` | `{{.OK}}`（または `{{.Criteria.OK}}`） |
| `{{.Criteria.ng}}` | `{{.NG}}`（または `{{.Criteria.NG}}`） |
| `{{.Criteria.repeat}}` | `{{.Repeat}}`（または `{{.Criteria.Repeat}}`） |
| `{{index .Criteria "ok"}}` | `{{.OK}}` |
| `{{range $key, $values := .Criteria}}` | `{{.OK}}` / `{{.NG}}` / `{{.Repeat}}` を個別に使用 |
| `{{if .Criteria}}` | `{{if .HasCriteria}}`（推奨） |

### HTMLの自動エスケープ

テンプレートファイル**または**出力ファイルの拡張子が `.html` / `.htm` / `.xhtml`（大文字・小文字は区別しない）の場合，テンプレートはGoの[html/template](https://pkg.go.dev/html/template)で処理され，問題文などの値は自動的にエスケープされます．それ以外の拡張子（`.md`，`.csv`，`.txt`など）では[text/template](https://pkg.go.dev/text/template)で処理され，値はそのまま出力されます．

- 問題文に `<script>` などが含まれていても，HTMLテンプレートではタグとして解釈されずに文字として表示されます．
- テンプレートや関数の結果に含まれるHTMLもエスケープされます．例えば `{{join .Comments "<br>"}}` は改行にならず `&lt;br&gt;` と出力されます．意図的にHTMLとして出力したい場合は `safeHTML` を使ってください（ただし値自体もエスケープされなくなるため，信頼できるデータにのみ使用してください）．
- html/templateは出力位置の文脈に応じてエスケープ方法を変えるため，text/templateとは結果が異なる場合があります：
  - `<!-- -->` のHTMLコメントは出力から削除されます．
  - `<script>` 内の値はJavaScriptの文字列リテラルとして（`"`付きで）出力されます．
  - `href` などのURL属性に安全でない値を入れると `#ZgotmplZ` に置き換えられます．
- `{{if}}` の分岐ごとにタグや属性の開閉が揃っていないテンプレートはエラーになります．エラーの場合，既存の出力ファイルは変更されません．
- Markdownビューアの多くはMarkdown中のHTMLを描画するため，Markdownテンプレートでは `escapeHTML` を使って値をエスケープすることを推奨します（組み込みの `quiz_template.md` はそのようにしています）．

### テンプレート例

#### Markdownテンプレート

```markdown
# 自作問題

{{range $index, $item := .Items}}
## Q{{add $index 1}}

**Q**: {{escapeHTML .Question | nl2br}}
**A**: {{escapeHTML .Answer | nl2br}}
{{if .Spell}}**読み**: {{escapeHTML .Spell}}{{end}}
{{if .Comments}}**コメント**:
{{range .Comments}}
- {{escapeHTML .}}
{{end}}{{end}}
{{if .HasCriteria}}**判定**: {{formatCriteria .Criteria | escapeHTML}}{{end}}

---
{{end}}

総問題数: {{len .Items}}問
```

#### HTMLテンプレート
```html
<!DOCTYPE html>
<html>
<head><title>クイズ</title></head>
<body>
    <h1>自作問題</h1>
    {{range $index, $item := .Items}}
    <div class="quiz-item">
        <h3>Q{{add $index 1}}</h3>
        <p><strong>問題:</strong> {{.Question}}</p>
        <p><strong>答え:</strong> {{.Answer}}</p>
        {{if .Comments}}
        <div class="comments">
            <strong>コメント:</strong>
            <ul>
                {{range .Comments}}<li>{{.}}</li>{{end}}
            </ul>
        </div>
        {{end}}
        {{if .HasCriteria}}<p><strong>判定:</strong> {{formatCriteria .Criteria}}</p>{{end}}
    </div>
    {{end}}
</body>
</html>
```

#### テキストテンプレート
```
{{range $index, $item := .Items}}
問題{{add $index 1}}: {{.Question}}
答え: {{.Answer}}
{{if .Spell}}読み: {{.Spell}}{{end}}
{{if .HasCriteria}}判定: {{formatCriteria .Criteria}}{{end}}

{{end}}
```

#### CSVテンプレート

CSVもテンプレートで列の構成や順序を変えられます．値にカンマ・ダブルクォート・改行が含まれていても壊れないよう，`csvRowSafe`（または`csvFieldSafe`）を使って出力してください．

`csvRowSafe` / `csvFieldSafe` は，`=HYPERLINK(...)` や `@SUM(...)` のような値をExcelやGoogle スプレッドシートで開いたときに数式として実行されないよう（CSV injection対策），次のように出力します．

- `=` / `+` / `-` / `@` / タブ / CR で始まる値の先頭に `'` を付けます（例: `=1+2` → `'=1+2`，`- 項目` → `'- 項目`）．先頭の空白（全角スペースなどを含む）は無視して判定します
- `-1` / `+5` / `-1.5` のように数値だけからなる値は，数式として実行されないため，そのまま出力します
- `;` を含む値は，ダブルクォートで囲みます（`;` を区切り文字とする地域の表計算ソフトで，`a;=1+2` が別のセルに分かれて実行されないようにするため）
- 全角の `＝` などで始まる値は対象外です

表計算ソフト以外のプログラムで読み込むなど，値をそのまま出力したい場合は，`'` を付けない `csvRow` / `csvField` を使えます．ただし，他人が作ったYAMLを変換して表計算ソフトで開くと数式が実行されるおそれがあるため，信頼できるデータにだけ使ってください．ヘッダー行のような固定の文字列には `csvRow` を使って構いません．

`-template` で指定する自作のCSVテンプレートでは，この対策は `csvRowSafe` / `csvFieldSafe` を使った箇所にだけ効きます．以前のガイドの例のように `csvRow` を使っている場合は，`csvRowSafe` に置き換えてください．

`templates/quiz_template.csv` は，テンプレートを使わない場合のCSV出力に使われる組み込みテンプレートです（バイナリに埋め込まれているため，このファイルを編集しても再ビルドするまで `-format csv` の出力は変わりません）．列構成を変えたい場合は，このファイルをコピーして編集し，`-template` で指定してください．

```text
{{csvRow "question" "answer" "spell" "criteria"}}
{{range .Items}}{{csvRowSafe .Question .Answer .Spell (formatCriteria .Criteria)}}
{{end -}}
```

列を変えたい場合の例（番号・答え・別解・タグ）:

```text
{{csvRow "no" "answer" "ok" "tags"}}
{{range $i, $item := .Items}}{{csvRowSafe (printf "%d" (add $i 1)) .Answer (quoteAll .OK "") (join .Tags " ")}}
{{end -}}
```

- `{{end -}}` の `-` は，最後の行のあとに余分な空行が出力されないようにするためのものです．
- `csvRow` / `csvRowSafe` の引数はすべて文字列です．数値は `printf "%d"` で文字列に変換してください．

## 出力形式の自動判定

- テンプレートファイル指定時 → テンプレート形式で出力（拡張子が`.csv`でもテンプレートが優先されます）
- `.csv`拡張子 → 組み込みのCSV形式（`templates/quiz_template.csv`）で出力
- `-format html` / `-format markdown` → バイナリに埋め込まれた組み込みテンプレート（`templates/quiz_template.html` / `.md`）で出力（カレントディレクトリに関係なく使えます）
- その他の拡張子 → テンプレート形式で出力

## 実用例

### 1. LaTeX形式での出力
```bash
go run . -input quiz.yaml -output quiz.tex -template quiz_template.tex
```

### 2. JSON形式での出力
```bash
go run . -input quiz.yaml -output quiz.json -template json_template.json
```

### 3. プレーンテキスト
```bash
go run . -input quiz.yaml -output quiz.txt -template simple_template.txt
```

