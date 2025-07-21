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
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/m-uesaka/quiz-yaml-go/quiz_yaml_converter" // Import the quiz YAML converter package
)

func main() {
	// フラグの定義
	var (
		inputFile  = flag.String("input", "", "入力するYAMLファイルのパス（必須）")
		outputFile = flag.String("output", "", "出力ファイルのパス（必須）")
		format     = flag.String("format", "csv", "出力フォーマット（csv, html, markdown）")
		template   = flag.String("template", "", "テンプレートファイルのパス（formatに関係なく使用）")
		validate   = flag.Bool("validate", false, "YAMLファイルのフォーマットをバリデーションのみ実行")
		help       = flag.Bool("help", false, "ヘルプを表示")
	)

	// ヘルプメッセージをカスタマイズ
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "使用法: %s [オプション]\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "クイズYAMLファイルを指定されたフォーマットに変換します。\n\n")
		fmt.Fprintf(os.Stderr, "オプション:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n例:\n")
		fmt.Fprintf(os.Stderr, "  %s -input quiz.yaml -output quiz.csv\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "  %s -input quiz.yaml -output quiz.html -format html\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "  %s -input quiz.yaml -output quiz.md -template custom.tmpl\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "  %s -input quiz.yaml -validate\n", filepath.Base(os.Args[0]))
	}

	// フラグをパース
	flag.Parse()

	// ヘルプフラグがセットされている場合
	if *help {
		flag.Usage()
		return
	}

	// 必須パラメータの検証
	if *inputFile == "" {
		fmt.Fprintf(os.Stderr, "❌ エラー: 入力ファイルが指定されていません\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// バリデーションのみの場合
	if *validate {
		fmt.Printf("🔍 YAMLファイルをバリデーションしています: %s\n", *inputFile)
		result := quiz_yaml_converter.ValidateYAMLFile(*inputFile)

		if result.IsValid {
			fmt.Printf("✅ バリデーション成功: %d問のクイズデータが正しく読み込めました\n", result.Items)
		} else {
			fmt.Printf("❌ バリデーション失敗: %d個のエラーが見つかりました\n", len(result.Errors))
			for _, err := range result.Errors {
				fmt.Fprintf(os.Stderr, "  • %s\n", err)
			}
			os.Exit(1)
		}
		return
	}

	// 変換モードの場合は出力ファイルが必須
	if *outputFile == "" {
		fmt.Fprintf(os.Stderr, "❌ エラー: 出力ファイルが指定されていません\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// テンプレートファイルが指定されている場合はテンプレート変換を実行
	if *template != "" {
		err := quiz_yaml_converter.Convert(*inputFile, *outputFile, *template)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ エラー: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ テンプレート変換完了: %s + %s → %s\n", *inputFile, *template, *outputFile)
		return
	}

	// フォーマットに基づいて変換処理を実行
	switch *format {
	case "csv":
		err := quiz_yaml_converter.Convert(*inputFile, *outputFile, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ エラー: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ CSV変換完了: %s → %s\n", *inputFile, *outputFile)

	case "html":
		templatePath := "templates/quiz_template.html"
		err := quiz_yaml_converter.Convert(*inputFile, *outputFile, templatePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ エラー: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ HTML変換完了: %s → %s\n", *inputFile, *outputFile)

	case "markdown", "md":
		templatePath := "templates/quiz_template.md"
		err := quiz_yaml_converter.Convert(*inputFile, *outputFile, templatePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ エラー: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Markdown変換完了: %s → %s\n", *inputFile, *outputFile)

	default:
		fmt.Fprintf(os.Stderr, "❌ エラー: サポートされていないフォーマットです: %s\n", *format)
		fmt.Fprintf(os.Stderr, "サポートされているフォーマット: csv, html, markdown\n\n")
		flag.Usage()
		os.Exit(1)
	}
}
