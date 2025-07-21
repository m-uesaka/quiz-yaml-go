// YAMLフォーマットの問題データをテンプレート処理，もしくはCSV形式に変換するためのパッケージです．
// このパッケージが提供する機能は以下の通りです：
//   - YAMLファイルからの問題データの読み込み
//   - 日本語テキストを適切に処理したCSV形式への変換
//   - Goのテンプレートを使用したカスタムフォーマットへの変換
//   - 日本語クイズフォーマット用の組み込みテンプレート関数
package quiz_yaml_converter

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

// 1問ごとのエントリを表す構造体
// 問題文、答え、原語表記、コメント、および判定基準を含む。
type QuizItem struct {
	Question string              `yaml:"question"`           // 問題文
	Answer   string              `yaml:"answer"`             // 答え
	Spell    string              `yaml:"spell"`              // 原語表記（英語表記）
	Comments []string            `yaml:"comments,omitempty"` // コメント
	Criteria map[string][]string `yaml:"criteria,omitempty"` // 判定基準（ok/ng/repeat）
}

// テンプレート処理用のデータ構造体
// 問題データのリストを含む。
type TemplateData struct {
	Items []QuizItem // 問題データのリスト
}

// 出力される文字列
type OutputFormat string

// 出力形式
const (
	FormatCSV      OutputFormat = "csv"      // CSV形式
	FormatTemplate OutputFormat = "template" // テンプレート形式
)

// 必要に応じて「」を追加する．
//
// - 既に"「"で始まり"」"で終わっている場合はそのまま返す
// - "「"で始まっているが"」"で終わっていない場合
//   - "」"がどこかに含まれている場合はそのまま返す
//   - 含まれていない場合は"」"を追加して返す
// - "」"を含むが"「"で始まっていない場合は"「"を追加して返す
// - どちらも含まれていない場合は"「"と"」"を最初と最後に追加して返す．
func AddQuotesIfNeeded(item string) string {
	// Already starts with 「 and ends with 」
	if strings.HasPrefix(item, "「") && strings.HasSuffix(item, "」") {
		return item
	}
	// Starts with 「 but doesn't end with 」
	if strings.HasPrefix(item, "「") {
		// Contains 」 somewhere (like 「美術館」（おまけ））
		if strings.Contains(item, "」") {
			return item
		} else {
			// Starts with 「 but no 」, add 」
			return item + "」"
		}
	}
	// Ends with 」 but doesn't start with 「
	if !strings.HasPrefix(item, "「") && strings.HasSuffix(item, "」") {
		return "「" + item
	}
	// No quotes at all, add both
	return "「" + item + "」"
}

// 正誤判定の文字列をフォーマットするための補助関数
// 別解などの単語に適切に「」を追加して羅列し，最後に指定された文字列を追加する．
func formatCriteriaSection(items []string, suffix string) string {
	if len(items) == 0 {
		return ""
	}

	var formattedItems []string
	for _, item := range items {
		formattedItems = append(formattedItems, AddQuotesIfNeeded(item))
	}
	return strings.Join(formattedItems, "") + suffix
}

// 正誤判定のフォーマットを行う．
//
// 形式は「別解1」「別解2」／「誤答1」は誤答／「もう一度1」はもう一度という形式で返す．
func FormatCriteria(criteria map[string][]string) string {
	var parts []string

	// Process ok
	if ok, exists := criteria["ok"]; exists && len(ok) > 0 {
		parts = append(parts, formatCriteriaSection(ok, ""))
	}

	// Process ng
	if ng, exists := criteria["ng"]; exists && len(ng) > 0 {
		parts = append(parts, formatCriteriaSection(ng, "は誤答"))
	}

	// Process repeat
	if repeat, exists := criteria["repeat"]; exists && len(repeat) > 0 {
		parts = append(parts, formatCriteriaSection(repeat, "はもう一度"))
	}

	return strings.Join(parts, "／")
}

// 出力されるファイルのフォーマットを返す．
// テンプレートファイルが指定されている場合はFormatTemplateを返し，
// それ以外は出力ファイルの拡張子からフォーマットを検出する．
func DetectOutputFormat(outputFile, templateFile string) OutputFormat {
	if templateFile != "" {
		return FormatTemplate
	}

	ext := strings.ToLower(filepath.Ext(outputFile))
	if ext == ".csv" {
		return FormatCSV
	}

	return FormatTemplate
}

// YAMLファイルからデータを読み込む．
func LoadYAMLData(yamlFilePath string) ([]QuizItem, error) {
	yamlFile, err := os.Open(yamlFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open YAML file: %w", err)
	}
	defer yamlFile.Close()

	yamlData, err := io.ReadAll(yamlFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %w", err)
	}

	var data []QuizItem
	err = yaml.Unmarshal(yamlData, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return data, nil
}
// 問題データとテンプレートファイルから出力ファイルを生成する．
// テンプレートはGoのtext/templateパッケージを使用し，日本語クイズフォーマット用のカスタム関数を提供する．
func ConvertToTemplate(data []QuizItem, templateFilePath, outputFilePath string) error {
	// Read template file
	templateContent, err := os.ReadFile(templateFilePath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	// Create template with custom functions
	tmpl, err := template.New("quiz").Funcs(template.FuncMap{
		"formatCriteria": FormatCriteria,
		"addQuotes":      AddQuotesIfNeeded,
		"join":           strings.Join,
		"upper":          strings.ToUpper,
		"lower":          strings.ToLower,
		"replace":        strings.ReplaceAll,
		"add": func(a, b int) int {
			return a + b
		},
		"len": func(slice []QuizItem) int {
			return len(slice)
		},
		"now": func() string {
			return time.Now().Format("2006年01月02日 15:04:05")
		},
	}).Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create output file
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Execute template
	templateData := TemplateData{Items: data}
	err = tmpl.Execute(outputFile, templateData)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// YAMLファイルをCSVファイルに変換する．
// CSV出力には問題文、答え、原語表記、およびフォーマットされた正誤判定が含まれる．
func ConvertYAMLToCSV(yamlFilePath, csvFilePath string) error {
	data, err := LoadYAMLData(yamlFilePath)
	if err != nil {
		return err
	}

	// Create CSV file
	csvFile, err := os.Create(csvFilePath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	// Write header
	err = writer.Write([]string{"question", "answer", "spell", "criteria"})
	if err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, item := range data {
		criteriaText := ""
		if item.Criteria != nil {
			criteriaText = FormatCriteria(item.Criteria)
		}

		err = writer.Write([]string{
			item.Question,
			item.Answer,
			item.Spell,
			criteriaText,
		})
		if err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// 全体の変換処理を行うエントリーポイント．
// 出力ファイルのフォーマットを検出し，CSV形式またはテンプレート形式に変換する．
// 出力ファイルの拡張子やテンプレートファイルの有無に基づいて適切な変換関数を呼び出す．
func Convert(yamlFilePath, outputFilePath, templateFilePath string) error {
	format := DetectOutputFormat(outputFilePath, templateFilePath)

	switch format {
	case FormatCSV:
		return ConvertYAMLToCSV(yamlFilePath, outputFilePath)
	case FormatTemplate:
		if templateFilePath == "" {
			return fmt.Errorf("template file is required for non-CSV output")
		}
		data, err := LoadYAMLData(yamlFilePath)
		if err != nil {
			return err
		}
		return ConvertToTemplate(data, templateFilePath, outputFilePath)
	default:
		return fmt.Errorf("unsupported output format")
	}
}
