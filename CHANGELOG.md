# 変更履歴

このプロジェクトの主な変更を記録します．
形式は [Keep a Changelog](https://keepachangelog.com/ja/1.1.0/) に，バージョン番号は [セマンティック バージョニング](https://semver.org/lang/ja/) に従います．

- **メジャーバージョン**（例: 1.0.0 → 2.0.0）：YAMLの形式，テンプレートの書き方，CLIの引数，Goのパッケージの公開APIのいずれかに，互換性のない変更がある
- **マイナーバージョン**（例: 1.0.0 → 1.1.0）：互換性を保ったまま機能を追加する
- **パッチバージョン**（例: 1.0.0 → 1.0.1）：互換性を保ったままバグを修正する

各バージョンで配布しているバイナリは [Releases](https://github.com/m-uesaka/quiz-yaml-go/releases) から取得できます．

## [Unreleased]

### 変更

- `-markdown-dir` で `-recursive` を指定しない場合も，拡張子の大文字・小文字を区別せずにMarkdownファイルを集めるようにした（#19）
  - `QUIZ.MD` や `quiz.Md` も対象になる（`-recursive` を指定した場合は以前からこの動作だった）
  - 拡張子が `.md` のディレクトリと，ディレクトリを指すシンボリックリンクは対象外になる
- `-markdown-dir` に指定したディレクトリにMarkdownファイルが1つもない場合，空のリストを書き出さずにエラーにするようにした（#30）
  - パスの指定ミスで既存の出力ファイルが空のリストに上書きされることを防ぐため
  - ライブラリの `AggregateMarkdownDir` も，空のスライスではなくエラーを返す
- `-markdown-dir` にファイルを指定した場合，`-recursive` の有無にかかわらずエラーにするようにした（#30）
  - 以前は `-recursive` を指定するとそのファイル1つだけを変換していた

### 修正

- `-markdown-dir` に存在しないディレクトリを指定した場合，`-recursive` の有無にかかわらずエラーになるようにした（#19, #30）
- `-markdown-dir` にディレクトリを指すシンボリックリンクを指定して `-recursive` を付けた場合，リンク先のファイルが集められずに空のリストが書き出される問題を修正（#30）
- `criteria` に `<<: *anchor` で取り込んだ内容に `ok` / `ng` / `repeat` 以外のキー（書き間違いなど）がある場合，エラーにならずに無視されていた問題を修正（#37）
  - エラーには取り込み元の行番号を `（3行目，<< で取り込み）` のように表示する
- 先頭にUTF-8のBOMが付いたMarkdownファイル（Windowsのメモ帳などで保存したもの）が「frontmatterが見つかりません」で失敗する問題を修正（#19）

## [1.0.0] - 2026-11-04

判定基準（criteria）の型の変更など，テンプレートの書き方に互換性のない変更があります．
v0.x から更新する場合は，[v0.x から v1.0.0 への移行](#v0x-から-v100-への移行) を確認してください．

### 破壊的変更

- `QuizItem.Criteria` の型を `map[string][]string` から専用の構造体 `*Criteria` に変更（#17, #33）
  - テンプレートで `{{.Criteria.ok}}` や `{{index .Criteria "ok"}}` のようにキー名で参照していた箇所はエラーになる
  - Goのパッケージから利用している場合，`FormatCriteria(map[string][]string)` 関数は削除され，`(*Criteria).Format()` メソッドに置き換わった
- HTMLテンプレート（拡張子が `.html` / `.htm` / `.xhtml`）では，値を自動的にエスケープするようになった（#6, #8）
  - 問題文などに書いたHTMLタグは，タグとして解釈されずに文字として出力される
- criteria に `ok` / `ng` / `repeat` 以外のキーがある場合，変換がエラーになる（#17, #33）
  - 以前は変換時に黙って無視されていた

### 追加

- テンプレートから判定基準に個別にアクセスする `{{.OK}}` / `{{.NG}}` / `{{.Repeat}}` と，テンプレート関数 `quoteAll`（#1, #3）
- 判定基準の有無を判定する `{{.HasCriteria}}`（#33）
  - `criteria: {}` のように中身が空の場合も偽になる
- CSVをテンプレートから生成する機能と，テンプレート関数 `csvRow` / `csvField`，組み込みテンプレート `templates/quiz_template.csv`（#2, #9）
- HTMLのエスケープを制御するテンプレート関数 `escapeHTML` / `safeHTML`（#6, #8）
- `-validate` で，criteria の不正なキーを行番号付きで報告する（#33）
  - 例: `問題 1: 不正なcriteriaキー: 'maybe'（5行目） (使用可能: ok, ng, repeat)`

### 変更

- テンプレート関数 `len` を独自の実装からGo標準の `len` に変更（#3）
  - `{{len .Items}}` は従来どおり使え，`{{len .OK}}` のようにスライス全般に使えるようになった
- 組み込みの HTML / Markdown テンプレートで，判定基準の有無の判定を `{{if .Criteria}}` から `{{if .HasCriteria}}` に変更（#33）
- `-markdown-dir` による Markdown → YAML 変換で書き出す criteria のキーの順序を，`ng, ok, repeat` から `ok, ng, repeat` に変更（#33）
  - キー名と形式は変わらないが，以前に生成したYAMLと比べると差分が出る

### 修正

- テンプレートの実行が途中で失敗した場合に，既存の出力ファイルが書きかけの内容で上書きされないようにした（#9）
- テンプレートとCSVの改行コードをLFに固定し，Windowsでチェックアウトしても出力が変わらないようにした（#9）

### 開発環境

- PR と main への push でフォーマットチェック・静的解析・テストを実行するCIを追加し，タスクランナー（Taskfile）を導入（#13, #26）
- テンプレート関数・アクセサのテストと，組み込みテンプレートのスナップショットテストを追加（#7, #27）
- CLIの処理を `run` 関数に切り出してテストを追加（#16, #28）．CLIの動作は変わらない

## [0.1.0] - 2026-07-15

### 追加

- Markdownファイル（1問1ファイル）を1つのYAMLファイルに集約する機能（`-markdown-dir` / `-recursive`）
- `QuizItem` の `tags` フィールド（Markdownの frontmatter の `tags` をYAMLに反映する）

## [0.0.1] - 2026-01-31

最初のリリース．

### 追加

- YAMLの問題データをCSVに変換する機能
- Goのテンプレートを使って HTML / Markdown などに変換する機能と，組み込みテンプレート
- YAMLファイルのバリデーション（`-validate`）

---

## v0.x から v1.0.0 への移行

YAMLファイルの形式は変わりません．自作のテンプレートを使っている場合と，Goのパッケージとして利用している場合に対応が必要です．

### 1. テンプレートでの判定基準の参照

キー名（小文字）での参照を，以下のように書き換えてください．

| v0.x の書き方 | v1.0.0 の書き方 |
|---------------|-----------------|
| `{{.Criteria.ok}}` | `{{.OK}}`（または `{{.Criteria.OK}}`） |
| `{{.Criteria.ng}}` | `{{.NG}}`（または `{{.Criteria.NG}}`） |
| `{{.Criteria.repeat}}` | `{{.Repeat}}`（または `{{.Criteria.Repeat}}`） |
| `{{index .Criteria "ok"}}` | `{{.OK}}` |
| `{{range $key, $values := .Criteria}}` | `{{.OK}}` / `{{.NG}}` / `{{.Repeat}}` を個別に使う |
| `{{if .Criteria}}` | `{{if .HasCriteria}}`（推奨） |

- `{{formatCriteria .Criteria}}` はそのまま使えます
- `{{.Criteria.OK}}` は，判定基準が無い問題では `.Criteria` が `nil` のためエラーになります．`{{if .HasCriteria}}` で囲むか，`{{.OK}}` を使ってください
- `{{if .Criteria}}` もそのまま使えますが，`criteria: {}` のように中身が空の場合に真になります（v0.x では偽）

### 2. HTMLテンプレートでのエスケープ

拡張子が `.html` / `.htm` / `.xhtml` のテンプレート，または出力ファイルでは，値が自動的にエスケープされます．
問題文やコメントにHTMLタグを書いていて，タグとして出力したい場合は `safeHTML` を使ってください（信頼できる値にのみ使用してください）．

```html
<!-- v0.x: タグがそのまま出力されていた -->
<p>{{.Question}}</p>

<!-- v1.0.0: タグとして出力したい場合 -->
<p>{{safeHTML .Question}}</p>
```

### 3. criteria の不正なキー

criteria に `ok` / `ng` / `repeat` 以外のキー（`OK` のような大文字・小文字の違いを含む）があると，変換がエラーになります．
`-validate` で該当する箇所を確認し，正しいキー名に直すか削除してください．

```bash
./quiz-yaml-converter -input quiz.yaml -validate
```

なお，問題のフィールド（`question` など）に未定義のキーを書いた場合は，v0.x と同じくエラーにならずに無視されます．

### 4. Goのパッケージとして利用している場合

| v0.x | v1.0.0 |
|------|--------|
| `QuizItem{Criteria: map[string][]string{"ok": {"a"}}}` | `QuizItem{Criteria: &Criteria{OK: []string{"a"}}}` |
| `item.Criteria["ok"]` | `item.OK()`（`Criteria` が `nil` でも安全）または `item.Criteria.OK` |
| `FormatCriteria(item.Criteria)` | `item.Criteria.Format()`（`nil` の場合は空文字を返す） |
| `item.Criteria != nil && len(item.Criteria) > 0` | `item.HasCriteria()` |

詳しくは [テンプレートの作成方法](templates/TEMPLATE_GUIDE.md) と [YAMLファイルの作成方法](yaml/YAML_GUIDE.md) を参照してください．

[Unreleased]: https://github.com/m-uesaka/quiz-yaml-go/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/m-uesaka/quiz-yaml-go/compare/v0.1.0...v1.0.0
[0.1.0]: https://github.com/m-uesaka/quiz-yaml-go/compare/v0.0.1...v0.1.0
[0.0.1]: https://github.com/m-uesaka/quiz-yaml-go/releases/tag/v0.0.1
