# Quiz YAML Go

YAMLファイルからクイズデータを読み込み、CSV、HTML、Markdownなどの各種フォーマットに変換するGoアプリケーション

## 開発環境

- Go 1.24.5
- gopkg.in/yaml.v3 v3.0.1

## 使用方法

### go runを使う方法

```bash
# CSV形式に変換
go run main.go -input quiz.yaml -output quiz.csv

# HTML形式に変換
go run main.go -input quiz.yaml -output quiz.html -format html

# Markdown形式に変換
go run main.go -input quiz.yaml -output quiz.md -format markdown

# カスタムテンプレートを使用
go run main.go -input quiz.yaml -output custom.html -template my_template.html

# Markdownディレクトリを1つのYAMLファイルに集約
go run main.go -markdown-dir path/to/quiz -output quiz.yaml

# サブディレクトリも再帰的に辿って集約
go run main.go -markdown-dir path/to/quiz -recursive -output quiz.yaml
```

### ビルドして使う方法

```bash
# アプリケーションをビルド
go build -o quiz-yaml-converter

# ビルドしたバイナリを使用
./quiz-yaml-converter -input quiz.yaml -output quiz.csv
./quiz-yaml-converter -input quiz.yaml -output quiz.html -format html
```

### テストの実行方法

```bash
# 全てのテストを実行
go test ./...

# 特定のパッケージのテストを実行
go test ./quiz_yaml_converter

# カバレッジ付きでテストを実行
go test -cover ./...

# 詳細な出力でテストを実行
go test -v ./...
```

### Taskfileを使う方法

[Task](https://taskfile.dev/) をインストールしていれば、`Taskfile.yml` に定義したタスクを実行できます。
CI（`.github/workflows/ci.yml`）も同じタスクを実行しています。

```bash
# タスクの一覧を表示
task

# CIと同じチェック（gofmt・go mod tidy・go vet・go test）をまとめて実行
task ci

# 個別に実行
task fmt-check       # gofmtで整形されていないファイルがあれば失敗
task vet             # go vet ./...
task mod-tidy-check  # go.mod / go.sum が go mod tidy 済みでなければ失敗
task test            # go test -race -cover ./...

# gofmtでコードを整形
task fmt
```

### YAMLファイルのバリデーション

```bash
# YAMLファイルの構文と内容をチェック（変換は行わない）
go run main.go -input quiz.yaml -validate

# ビルド済みバイナリを使用
./quiz-yaml-converter -input quiz.yaml -validate
```

`-validate` を指定しない通常の変換や `-markdown-dir` での集約でも同じ内容をチェックし，問題があれば標準エラー出力に警告を表示します．
警告があっても出力ファイルは書き出され，終了コードは0になります（criteria の不正なキーだけは，従来どおり変換エラーになります）．

```
⚠️ 警告: 問題 1: 答え (answer) が空です
✅ CSV変換完了: quiz.yaml → quiz.csv
```

`-markdown-dir` での警告の問題番号は，出力されるYAMLファイルでの順番（ファイル名順）を表します．

## ディレクトリ構造

```
quiz-yaml-go/
├── main.go                    # メインエントリーポイント
├── go.mod                     # Go modules設定ファイル
├── go.sum                     # 依存関係のチェックサム
├── README.md                  # プロジェクト説明（このファイル）
├── CHANGELOG.md               # 変更履歴とバージョン間の移行方法
├── Taskfile.yml               # タスクランナー（Task）の設定
├── .gitignore                 # Git除外設定
├── quiz_yaml_converter/       # クイズ変換ライブラリパッケージ
│   ├── doc.go                 # パッケージの説明
│   ├── model.go               # データモデル（QuizItem, TemplateData, OutputFormat）
│   ├── criteria.go            # 判定基準（Criteria）の読み込みとフォーマット
│   ├── csv.go                 # CSVの整形とYAML→CSV変換
│   ├── template.go            # テンプレート処理（HTMLモードの判定を含む）
│   ├── validate.go            # YAMLファイルのバリデーション
│   ├── yaml_io.go             # YAMLファイルの読み書き
│   ├── convert.go             # 出力形式の判定と変換のエントリーポイント（Markdown→YAML変換を含む）
│   ├── markdown_parser.go     # Markdown→QuizItem変換ロジック
│   ├── *_test.go              # 各ファイルに対応するテストファイル
│   └── testdata/              # テスト用のデータ（テンプレートのスナップショットなど）
└── templates/                 # テンプレートファイル用ディレクトリ
    ├── TEMPLATE_GUIDE.md      # テンプレート作成ガイド
    ├── quiz_template.html     # HTML出力用テンプレート
    └── quiz_template.md       # Markdown出力用テンプレート
```

## コマンドライン引数

| 引数 | 必須 | デフォルト値 | 説明 |
|------|------|-------------|------|
| `-input` | ✓*2 | - | 入力するYAMLファイルのパス |
| `-markdown-dir` | | - | 集約するMarkdownファイルが置かれたディレクトリのパス（指定時はMarkdown→YAML変換モードになる．`-input`とは同時指定不可） |
| `-recursive` | | `false` | `-markdown-dir`指定時，サブディレクトリも再帰的に辿るかどうか |
| `-output` | *1 | - | 出力ファイルのパス |
| `-format` | | `csv` | 出力フォーマット（`csv`, `html`, `markdown`） |
| `-template` | | - | テンプレートファイルのパス（指定時はformatより優先） |
| `-validate` | | `false` | YAMLファイルのバリデーションのみ実行（出力は行わない） |
| `-help` | | `false` | ヘルプメッセージを表示 |

*1: `-validate`フラグ使用時は不要
*2: `-markdown-dir`指定時は不要（むしろ同時指定はエラー）

### 使用例

```bash
# ヘルプを表示
./quiz-yaml-converter -help

# YAMLファイルのバリデーション（変換は行わない）
./quiz-yaml-converter -input quiz.yaml -validate

# 基本的なCSV変換
./quiz-yaml-converter -input data/quiz.yaml -output output/quiz.csv

# HTML形式で出力（formatオプションを指定）
./quiz-yaml-converter -input data/quiz.yaml -output output/quiz.html -format html

# カスタムテンプレートを使用
./quiz-yaml-converter -input data/quiz.yaml -output output/custom.txt -template templates/custom.tmpl

# テンプレートを使ってCSVの列構成をカスタマイズ
./quiz-yaml-converter -input data/quiz.yaml -output output/quiz.csv -template templates/quiz_template.csv

# Markdownディレクトリを1つのYAMLファイルに集約
./quiz-yaml-converter -markdown-dir data/quiz -output output/quiz.yaml

# サブディレクトリも再帰的に辿って集約
./quiz-yaml-converter -markdown-dir data/quiz -recursive -output output/quiz.yaml
```

## テンプレートファイルの書き方

カスタムテンプレートファイルの作成方法については、[templates/TEMPLATE_GUIDE.md](templates/TEMPLATE_GUIDE.md)を参照してください。

## YAMLファイルの作成方法

入力用のYAMLファイルの作成方法については、[yaml/YAML_GUIDE.md](yaml/YAML_GUIDE.md)を参照してください。

## 変更履歴

バージョンごとの変更点と、互換性のない変更がある場合の移行方法については、[CHANGELOG.md](CHANGELOG.md)を参照してください。
v0.x から v1.0.0 に更新する場合は、テンプレートの書き方が一部変わっているため、[移行方法](CHANGELOG.md#v0x-から-v100-への移行)を確認してください。
