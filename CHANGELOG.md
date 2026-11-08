# 変更履歴

このプロジェクトの主な変更を記録します．
形式は [Keep a Changelog](https://keepachangelog.com/ja/1.1.0/) に，バージョン番号は [セマンティック バージョニング](https://semver.org/lang/ja/) に従います．

- **メジャーバージョン**（例: 1.0.0 → 2.0.0）：YAMLの形式，テンプレートの書き方，CLIの引数，Goのパッケージの公開APIのいずれかに，互換性のない変更がある
- **マイナーバージョン**（例: 1.0.0 → 1.1.0）：互換性を保ったまま機能を追加する
- **パッチバージョン**（例: 1.0.0 → 1.0.1）：互換性を保ったままバグを修正する

各バージョンで配布しているバイナリは [Releases](https://github.com/m-uesaka/quiz-yaml-go/releases) から取得できます．

## [Unreleased]

### 追加

- 通常の変換と `-markdown-dir` での集約でもバリデーションを行い，問題があれば標準エラー出力に警告を表示するようにした（#18）
  - `-validate` と同じ内容（クイズデータが1件もない，空の問題文・答え，空の要素）をチェックする
  - 警告があっても出力ファイルは書き出され，終了コードは0になる
  - criteria の不正なキーは，従来どおり変換エラーになる
  - `-markdown-dir` での警告には，元になったMarkdownファイルのパスを表示する
  - 一部のMarkdownファイルの変換に失敗した場合も，変換できたファイルの警告をエラーと一緒に表示する
- Goのパッケージに，データを受け取ってバリデーションする `ValidateItems` と，構造化したエラー型 `ValidationError` を追加（#18）
  - `ValidationResult` に構造化したエラーの `Details` フィールドを追加（`Errors` は従来どおり）
  - 警告を返す `ConvertWithWarnings` / `ConvertMarkdownDirToYAMLWithWarnings` を追加
  - `ValidationError` に，元になったファイルのパスを表す `Source` フィールドを追加
- Goのパッケージに，組み込みテンプレートで変換する `ConvertWithBuiltinTemplate` / `ConvertWithBuiltinTemplateWithWarnings` と，組み込みテンプレートの種類を表す `BuiltinTemplate`（`BuiltinCSV` / `BuiltinHTML` / `BuiltinMarkdown`）を追加（#11）
- 組み込みテンプレートを埋め込んだ `templates` パッケージ（`templates.FS`）を追加（#11）
- 表計算ソフトで数式として解釈されないようにCSVのフィールドを整形するテンプレート関数 `csvRowSafe` / `csvFieldSafe` を追加（#12）
  - Goのパッケージにも，対応する `CSVRowSafe` / `CSVFieldSafe` と，値の先頭に `'` を付けるだけの `EscapeCSVFormula` を追加
  - 既存の `csvRow` / `csvField`（`CSVRow` / `CSVField`）の出力は変わらない

### 変更

- 組み込みテンプレート（`templates/quiz_template.csv` / `.html` / `.md`）を `go:embed` でバイナリに埋め込むようにした（#11）
  - `-format html` / `-format markdown` が，リポジトリのルート以外をカレントディレクトリにしても使えるようになった
  - `-template` で指定したテンプレートファイルは，従来どおり実行時にファイルシステムから読み込む
- テンプレートを使わないCSV出力（`-format csv`，`ConvertYAMLToCSV`）を，組み込みのCSVテンプレートで出力するようにした（#11）
  - 出力内容は変わらない
  - すべての出力を「メモリ上で組み立て，テンプレートの実行に成功した場合のみファイルに書き込む」処理に揃えた（書き込みの途中で失敗した場合も既存のファイルを壊さないようにした．#47）
  - CSVの出力ファイルを作成できない場合のエラーメッセージが `failed to create CSV file: ...` から `出力ファイルの書き込みに失敗しました: ...` に変わる（日本語のメッセージは #20）
- テンプレートで出力する場合に新しく作るファイルの権限を，`0644` から，CSV出力と同じ `0666` から umask を引いたものにした（#11）
  - umask が `022` の場合は従来どおり `0644` になる．テンプレートを使わないCSV出力の権限は変わらない
- バリデーション処理を整理し，空要素のチェックを共通化した（#18）
- 入力のYAMLファイルを開けない場合のエラーメッセージを `failed to open YAML file: ...` から `YAMLファイルの読み込みに失敗しました: ...` に変更した（#18, #20）
- `-markdown-dir` で `-recursive` を指定しない場合も，拡張子の大文字・小文字を区別せずにMarkdownファイルを集めるようにした（#19）
  - `QUIZ.MD` や `quiz.Md` も対象になる（`-recursive` を指定した場合は以前からこの動作だった）
  - 拡張子が `.md` のディレクトリと，ディレクトリを指すシンボリックリンクは対象外になる
- `-markdown-dir` に指定したディレクトリにMarkdownファイルが1つもない場合，空のリストを書き出さずにエラーにするようにした（#30）
  - パスの指定ミスで既存の出力ファイルが空のリストに上書きされることを防ぐため
  - ライブラリの `AggregateMarkdownDir` も，空のスライスではなくエラーを返す
- `-markdown-dir` にファイルを指定した場合，`-recursive` の有無にかかわらずエラーにするようにした（#30）
  - 以前は `-recursive` を指定するとそのファイル1つだけを変換していた
- Goのパッケージが返すエラーメッセージのうち，英語だったものを日本語に統一した（#20）
  - CLIの表示や，すでに日本語だったメッセージ（Markdownの解析・バリデーションなど）に合わせるため
  - `failed to read YAML file` → `YAMLファイルの読み込みに失敗しました`，`failed to parse YAML` → `YAMLの解析に失敗しました`，`failed to marshal YAML` → `YAMLへの変換に失敗しました`，`failed to write YAML file` → `YAMLファイルの書き込みに失敗しました`，`invalid criteria keys` → `criteriaに不正なキーがあります`
  - `failed to read template file` → `テンプレートファイルの読み込みに失敗しました`，`failed to parse template` → `テンプレートの解析に失敗しました`，`failed to execute template` → `テンプレートの実行に失敗しました`，`failed to write output file` → `出力ファイルの書き込みに失敗しました`
  - `failed to read builtin template` → `組み込みテンプレートの読み込みに失敗しました`，`unsupported builtin template` → `未対応の組み込みテンプレートです`，`template file is required for non-CSV output` → `CSV以外の形式で出力するにはテンプレートファイルが必要です`，`unsupported output format` → `未対応の出力形式です`
  - `failed to read markdown file` → `Markdownファイルの読み込みに失敗しました`，`failed to parse frontmatter` → `frontmatterの解析に失敗しました`
  - 包んでいる元のエラー（Goの標準ライブラリや yaml.v3 のメッセージ）はそのまま．`errors.Is` / `errors.As` で `fs.ErrNotExist` などを判定できることも変わらない

### 修正

- CSV出力で書き込みやファイルのクローズに失敗した場合に，最後の行が失われてもエラーにならず「✅ CSV変換完了」と表示される問題を修正（#11）
- 出力ファイルの書き込みの途中で失敗した場合（ディスクが一杯，I/Oエラーなど）に，既存の出力ファイルが空になったり途中まで書き込まれたファイルが残ったりする問題を修正（#47）
  - テンプレートでの出力（CSV・HTML・Markdownを含む）と `-markdown-dir` のYAMLの出力で，出力先と同じディレクトリの一時ファイルに書き込んでから置き換えるようにした．失敗した場合は一時ファイルを削除する
  - 新しく作るファイルの権限は従来どおり（テンプレートでの出力は `0666`，YAMLの出力は `0644` から umask を引いたもの）
  - 既存のファイルを置き換える場合は，元のファイルの権限を引き継ぐ．所有者・グループとハードリンクは引き継がない
  - 出力先がシンボリックリンクの場合は，従来どおりリンク先のファイルを置き換え，リンクは残す
  - 書き込む権限の無い既存のファイル（読み取り専用にしたファイルなど）は，従来どおりエラーになる（置き換えない）
  - 出力先が通常のファイル以外（`/dev/stdout` などのデバイスファイルや名前付きパイプ）の場合と，既存のファイルには書き込めるがディレクトリに一時ファイルを作れない場合は，従来どおりそのまま書き込む
  - 書き込みの途中でプロセスが強制終了された場合（Ctrl-Cなど）は，一時ファイル（`.quiz-yaml-*.tmp`）が残ることがある
  - Windowsでは，出力先のファイルを他のプログラムで開いている場合に置き換えに失敗してエラーになることがある（既存のファイルはそのまま残る）
- `-markdown-dir` に存在しないディレクトリを指定した場合，`-recursive` の有無にかかわらずエラーになるようにした（#19, #30）
- `-markdown-dir` にディレクトリを指すシンボリックリンクを指定して `-recursive` を付けた場合，リンク先のファイルが集められずに空のリストが書き出される問題を修正（#30）
- `criteria` に `<<: *anchor` で取り込んだ内容に `ok` / `ng` / `repeat` 以外のキー（書き間違いなど）がある場合，エラーにならずに無視されていた問題を修正（#37）
  - エラーには取り込み元の行番号を `（3行目，<< で取り込み）` のように表示する
- CSV出力（`-format csv`，`ConvertYAMLToCSV`，組み込みのCSVテンプレート `templates/quiz_template.csv`）で，`=HYPERLINK(...)` や `@SUM(...)` のような値が，ExcelやGoogle スプレッドシートで開いたときに数式として実行される問題（CSV injection）を修正（#12）
  - 先頭の空白（全角スペースなどを含む）を除いた最初の文字が `=` / `+` / `-` / `@` / タブ / CR の値の先頭に `'` を付ける（OWASPの推奨する対策）．ただし `-1` / `+5` / `-1.5` のように数値だけからなる値はそのまま出力する
  - `;` を含む値はダブルクォートで囲む（`;` を区切り文字とする地域の表計算ソフトで，セルが分かれて数式が実行されないようにするため）
  - 他人が作ったYAMLを変換して開く場合にも安全なように，既定で有効にした
  - `- 項目` のように `-` などで始まる値は `'- 項目` のように出力が変わる．表計算ソフト以外のプログラムでCSVを読み込む場合など，以前と同じ出力が必要なときは，`csvRow` を使ったテンプレートを `-template` で指定する
  - `-template` で指定する自作のCSVテンプレートで `csvRow` / `csvField` を使っている場合は，この対策は効かない．`csvRowSafe` / `csvFieldSafe` に置き換える
- 先頭にUTF-8のBOMが付いたMarkdownファイル（Windowsのメモ帳などで保存したもの）が「frontmatterが見つかりません」で失敗する問題を修正（#19）
- 組み込みのMarkdownテンプレート（`templates/quiz_template.md`）で，問題文・答え・コメント・判定基準に改行が含まれると，段落が途切れたり2行目以降が見出し・リストなどとして解釈されたりする問題を修正（#31）
  - 改行を `<br>` に置き換えて1行にまとめるテンプレート関数 `nl2br` を追加し，組み込みのMarkdownテンプレートで使うようにした
- CLIで `-format csv` を指定しても，出力ファイルの拡張子が `.csv` でないと `template file is required for non-CSV output` で失敗する問題を修正（#29）
  - `-format` を指定した場合は，出力ファイルの拡張子に関係なく指定したフォーマットで出力する
  - `-format` と `-template` をどちらも省略した場合は従来どおり，拡張子が `.csv` のときだけCSVで出力し，それ以外の拡張子ではエラーになる．エラーメッセージは `-format` または `-template` の指定を促す内容に変わり，ヘルプも表示する
  - これに合わせて，`-format` のヘルプに表示していた既定値 `csv` をなくし，省略した場合の動作を説明するようにした．指定できる値に `md`（`markdown` と同じ）も表示する
  - 修正したのはCLIのみ．Goのパッケージの `Convert` は従来どおり，テンプレートを指定しない場合は出力ファイルの拡張子で形式を判定する．拡張子に関係なくCSVを出力する場合は `ConvertWithBuiltinTemplate(yamlFilePath, outputFilePath, BuiltinCSV)` を使う

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
