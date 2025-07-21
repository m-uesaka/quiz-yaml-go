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

## ディレクトリ構造

```
quiz-yaml-go/
├── main.go                    # メインエントリーポイント
├── go.mod                     # Go modules設定ファイル
├── go.sum                     # 依存関係のチェックサム
├── README.md                  # プロジェクト説明（このファイル）
├── .gitignore                 # Git除外設定
├── quiz_yaml_converter/       # クイズ変換ライブラリパッケージ
│   ├── converter.go           # メイン変換ロジック
│   └── converter_test.go      # テストファイル
└── templates/                 # テンプレートファイル用ディレクトリ
    ├── TEMPLATE_GUIDE.md      # テンプレート作成ガイド
    ├── quiz_template.html     # HTML出力用テンプレート
    └── quiz_template.md       # Markdown出力用テンプレート
```

## コマンドライン引数

| 引数 | 必須 | デフォルト値 | 説明 |
|------|------|-------------|------|
| `-input` | ✓ | - | 入力するYAMLファイルのパス |
| `-output` | ✓ | - | 出力ファイルのパス |
| `-format` | | `csv` | 出力フォーマット（`csv`, `html`, `markdown`） |
| `-template` | | - | テンプレートファイルのパス（指定時はformatより優先） |
| `-help` | | `false` | ヘルプメッセージを表示 |

### 使用例

```bash
# ヘルプを表示
./quiz-yaml-converter -help

# 基本的なCSV変換
./quiz-yaml-converter -input data/quiz.yaml -output output/quiz.csv

# HTML形式で出力
./quiz-yaml-converter -input data/quiz.yaml -output output/quiz.html -format html

# カスタムテンプレートを使用
./quiz-yaml-converter -input data/quiz.yaml -output output/custom.txt -template templates/custom.tmpl
```

## テンプレートファイルの書き方

カスタムテンプレートファイルの作成方法については、[templates/TEMPLATE_GUIDE.md](templates/TEMPLATE_GUIDE.md)を参照してください。

## YAMLファイルの作成方法

入力用のYAMLファイルの作成方法については、[yaml/YAML_GUIDE.md](yaml/YAML_GUIDE.md)を参照してください。
