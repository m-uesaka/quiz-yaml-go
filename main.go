// Package main provides a command-line tool for converting quiz YAML files to various output formats.
//
// This tool supports converting YAML quiz data to:
//   - CSV format for spreadsheet applications
//   - HTML format using built-in templates
//   - Markdown format for documentation
//   - Custom formats using user-provided templates
//
// Usage:
//
//	converter -input quiz.yaml -output quiz.csv
//	converter -input quiz.yaml -output quiz.html -format html
//	converter -input quiz.yaml -output quiz.md -format markdown
//	converter -input quiz.yaml -output custom.html -template my_template.html
//
// YAMLフォーマットを変換するメインスクリプト
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/m-uesaka/quiz-yaml-go/quiz_yaml_converter" // Import the quiz YAML converter package
)

// outputFormat は -format で指定できる出力フォーマット1つ分の設定を表す
type outputFormat struct {
	// template は組み込みテンプレートのパス．
	// 空文字の場合はテンプレートを指定せずにConvertを呼ぶ．Convertは出力ファイルの拡張子で形式を判定するため，
	// 拡張子が.csvのときだけCSVを出力し，それ以外の拡張子ではエラーになる（#29）．
	template string
	// label は完了メッセージに表示するフォーマット名
	label string
}

// outputFormats は -format の値と出力フォーマットの対応表
var outputFormats = map[string]outputFormat{
	"csv":      {template: "", label: "CSV"},
	"html":     {template: "templates/quiz_template.html", label: "HTML"},
	"markdown": {template: "templates/quiz_template.md", label: "Markdown"},
	"md":       {template: "templates/quiz_template.md", label: "Markdown"},
}

func main() {
	os.Exit(run(filepath.Base(os.Args[0]), os.Args[1:], os.Stdout, os.Stderr))
}

// run はコマンドライン引数args（プログラム名を除く）を解析して処理を実行し，終了コードを返す．
// progはヘルプメッセージに表示するプログラム名．
func run(prog string, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet(prog, flag.ContinueOnError)
	fs.SetOutput(stderr)

	// フラグの定義
	var (
		inputFile   = fs.String("input", "", "入力するYAMLファイルのパス（-markdown-dir未指定時は必須）")
		markdownDir = fs.String("markdown-dir", "", "集約するMarkdownファイルが置かれたディレクトリのパス（指定時はMarkdown→YAML変換モードになる）")
		recursive   = fs.Bool("recursive", false, "-markdown-dir指定時，サブディレクトリも再帰的に辿るかどうか")
		outputFile  = fs.String("output", "", "出力ファイルのパス（必須）")
		format      = fs.String("format", "csv", "出力フォーマット（csv, html, markdown）")
		template    = fs.String("template", "", "テンプレートファイルのパス（formatに関係なく使用）")
		validate    = fs.Bool("validate", false, "YAMLファイルのフォーマットをバリデーションのみ実行")
		help        = fs.Bool("help", false, "ヘルプを表示")
	)

	// ヘルプメッセージをカスタマイズ
	fs.Usage = func() {
		fmt.Fprintf(stderr, "使用法: %s [オプション]\n\n", prog)
		fmt.Fprintf(stderr, "クイズYAMLファイルを指定されたフォーマットに変換します。\n\n")
		fmt.Fprintf(stderr, "オプション:\n")
		fs.PrintDefaults()
		fmt.Fprintf(stderr, "\n例:\n")
		fmt.Fprintf(stderr, "  %s -input quiz.yaml -output quiz.csv\n", prog)
		fmt.Fprintf(stderr, "  %s -input quiz.yaml -output quiz.html -format html\n", prog)
		fmt.Fprintf(stderr, "  %s -input quiz.yaml -output quiz.md -template custom.tmpl\n", prog)
		fmt.Fprintf(stderr, "  %s -input quiz.yaml -output quiz.csv -template templates/quiz_template.csv\n", prog)
		fmt.Fprintf(stderr, "  %s -input quiz.yaml -validate\n", prog)
		fmt.Fprintf(stderr, "  %s -markdown-dir path/to/quiz -output quiz.yaml\n", prog)
		fmt.Fprintf(stderr, "  %s -markdown-dir path/to/quiz -recursive -output quiz.yaml\n", prog)
	}

	// usageError は引数のエラーmsgを表示し，空行を挟んでヘルプを表示して，終了コード1を返す
	usageError := func(msg string) int {
		fmt.Fprintf(stderr, "❌ エラー: %s\n\n", msg)
		fs.Usage()
		return 1
	}

	// failure は処理中のエラーを表示し，終了コード1を返す
	failure := func(err error) int {
		fmt.Fprintf(stderr, "❌ エラー: %v\n", err)
		return 1
	}

	// フラグをパース（エラーメッセージとヘルプはflagパッケージが表示する）
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	// ヘルプフラグがセットされている場合
	if *help {
		fs.Usage()
		return 0
	}

	// Markdown→YAML集約モードの場合
	if *markdownDir != "" {
		if *inputFile != "" {
			return usageError("-markdown-dirと-inputは同時に指定できません")
		}
		if *outputFile == "" {
			return usageError("出力ファイルが指定されていません")
		}
		if err := quiz_yaml_converter.ConvertMarkdownDirToYAML(*markdownDir, *outputFile, *recursive); err != nil {
			return failure(err)
		}
		fmt.Fprintf(stdout, "✅ Markdown集約完了: %s → %s\n", *markdownDir, *outputFile)
		return 0
	}

	// 必須パラメータの検証
	if *inputFile == "" {
		return usageError("入力ファイルが指定されていません")
	}

	// バリデーションのみの場合
	if *validate {
		fmt.Fprintf(stdout, "🔍 YAMLファイルをバリデーションしています: %s\n", *inputFile)
		result := quiz_yaml_converter.ValidateYAMLFile(*inputFile)

		if !result.IsValid {
			fmt.Fprintf(stdout, "❌ バリデーション失敗: %d個のエラーが見つかりました\n", len(result.Errors))
			for _, err := range result.Errors {
				fmt.Fprintf(stderr, "  • %s\n", err)
			}
			return 1
		}
		fmt.Fprintf(stdout, "✅ バリデーション成功: %d問のクイズデータが正しく読み込めました\n", result.Items)
		return 0
	}

	// 変換モードの場合は出力ファイルが必須
	if *outputFile == "" {
		return usageError("出力ファイルが指定されていません")
	}

	// テンプレートファイルが指定されている場合はテンプレート変換を実行
	if *template != "" {
		if err := quiz_yaml_converter.Convert(*inputFile, *outputFile, *template); err != nil {
			return failure(err)
		}
		fmt.Fprintf(stdout, "✅ テンプレート変換完了: %s + %s → %s\n", *inputFile, *template, *outputFile)
		return 0
	}

	// フォーマットに基づいて変換処理を実行
	f, ok := outputFormats[*format]
	if !ok {
		return usageError(fmt.Sprintf("サポートされていないフォーマットです: %s\nサポートされているフォーマット: csv, html, markdown", *format))
	}
	if err := quiz_yaml_converter.Convert(*inputFile, *outputFile, f.template); err != nil {
		return failure(err)
	}
	fmt.Fprintf(stdout, "✅ %s変換完了: %s → %s\n", f.label, *inputFile, *outputFile)
	return 0
}
