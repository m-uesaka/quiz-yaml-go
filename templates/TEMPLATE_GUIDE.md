# テンプレートの作成方法

## テンプレート機能

### 利用可能なデータ構造
```go
type TemplateData struct {
    Items []QuizItem  // クイズデータのスライス
}

type QuizItem struct {
    Question string             // 問題文
    Answer   string             // 答え
    Spell    string             // 原語表記（英語表記）
    Comments []string           // コメント（補足説明など）
    Criteria map[string][]string // 判定基準（ok/ng/repeat）
}
```

### 利用可能なテンプレート関数

| 関数名 | 説明 | 使用例 |
|--------|------|--------|
| `formatCriteria` | criteriaを専用形式でフォーマット | `{{formatCriteria .Criteria}}` |
| `addQuotes` | 「」引用符を追加 | `{{addQuotes .Answer}}` |
| `join` | 文字列スライスを結合 | `{{join .Strings ","}}` |
| `upper` | 大文字に変換 | `{{upper .Question}}` |
| `lower` | 小文字に変換 | `{{lower .Answer}}` |
| `replace` | 文字列置換 | `{{replace .Text "old" "new"}}` |
| `add` | 数値の加算 | `{{add $index 1}}` |
| `len` | スライスの長さ | `{{len .Items}}` |
| `now` | 現在日時 | `{{now}}` |

#### 注意
- `add`は数値の加算に使います．デフォルトでは`$index`は0から始まるため、1を加えることで1から始まる番号付けが可能です。
- `formatCriteria`は判定基準を以下のようにフォーマットします:

```text
「別解1」「別解2」／「誤答1」「誤答2」は誤答／「もう一度1」「もう一度2」はもう一度
```
  - 別解は`ok`，誤答は`ng`，もう一度は`repeat`キーの値を使用します．

### テンプレート例

#### Markdownテンプレート

```markdown
# クイズ問題集

{{range $index, $item := .Items}}
## 問題{{add $index 1}}

**Q**: {{.Question}}
**A**: {{.Answer}}
{{if .Spell}}**読み**: {{.Spell}}{{end}}
{{if .Comments}}**コメント**:
{{range .Comments}}
- {{.}}
{{end}}{{end}}
{{if .Criteria}}**判定**: {{formatCriteria .Criteria}}{{end}}

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
    <h1>クイズ問題集</h1>
    {{range $index, $item := .Items}}
    <div class="quiz-item">
        <h3>問題{{add $index 1}}</h3>
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
        {{if .Criteria}}<p><strong>判定:</strong> {{formatCriteria .Criteria}}</p>{{end}}
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
{{if .Criteria}}判定: {{formatCriteria .Criteria}}{{end}}

{{end}}
```

## 出力形式の自動判定

- `.csv`拡張子 → CSV形式で出力
- テンプレートファイル指定時 → テンプレート形式で出力  
- その他の拡張子 → テンプレート形式で出力

## 実用例

### 1. LaTeX形式での出力
```bash
go run converter.go quiz.yaml quiz.tex quiz_template.tex
```

### 2. JSON形式での出力
```bash
go run converter.go quiz.yaml quiz.json json_template.json
```

### 3. プレーンテキスト
```bash
go run converter.go quiz.yaml quiz.txt simple_template.txt
```

