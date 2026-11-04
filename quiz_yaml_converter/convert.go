package quiz_yaml_converter

import (
	"fmt"
	"path/filepath"
	"strings"
)

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

// 全体の変換処理を行うエントリーポイント．
// 出力ファイルのフォーマットを検出し，CSV形式またはテンプレート形式に変換する．
// 出力ファイルの拡張子やテンプレートファイルの有無に基づいて適切な変換関数を呼び出す．
// バリデーションの警告が必要な場合は ConvertWithWarnings を使う．
func Convert(yamlFilePath, outputFilePath, templateFilePath string) error {
	_, err := ConvertWithWarnings(yamlFilePath, outputFilePath, templateFilePath)
	return err
}

// ConvertWithWarnings は Convert と同じ変換を行い，読み込んだデータを ValidateItems で
// バリデーションした結果を警告として返す．警告があっても変換は中断しない．
// 警告は読み込みに成功した場合に返し，その後の変換が失敗した場合も返す．
// criteria の不正なキーは警告ではなく，従来どおりエラーになる．
func ConvertWithWarnings(yamlFilePath, outputFilePath, templateFilePath string) ([]ValidationError, error) {
	format := DetectOutputFormat(outputFilePath, templateFilePath)
	if format == FormatTemplate && templateFilePath == "" {
		return nil, fmt.Errorf("template file is required for non-CSV output")
	}

	data, err := LoadYAMLData(yamlFilePath)
	if err != nil {
		return nil, err
	}
	warnings := ValidateItems(data)

	switch format {
	case FormatCSV:
		// ConvertYAMLToCSV はファイルパスを受け取るため，YAMLファイルをもう一度読み込む
		return warnings, ConvertYAMLToCSV(yamlFilePath, outputFilePath)
	case FormatTemplate:
		return warnings, ConvertToTemplate(data, templateFilePath, outputFilePath)
	default:
		return warnings, fmt.Errorf("unsupported output format")
	}
}

// ConvertMarkdownDirToYAML はMarkdownディレクトリを1つのYAMLファイルに集約する
// エントリーポイント．recursiveがtrueの場合はサブディレクトリも再帰的に辿る．
// バリデーションの警告が必要な場合は ConvertMarkdownDirToYAMLWithWarnings を使う．
func ConvertMarkdownDirToYAML(mdDirPath, yamlFilePath string, recursive bool) error {
	_, err := ConvertMarkdownDirToYAMLWithWarnings(mdDirPath, yamlFilePath, recursive)
	return err
}

// ConvertMarkdownDirToYAMLWithWarnings は ConvertMarkdownDirToYAML と同じ集約を行い，
// 集約したデータを ValidateItems でバリデーションした結果を警告として返す．
// 警告の問題番号は，出力するYAMLファイルでの順番（1始まり）を表す．
// 警告があってもYAMLファイルは書き出す．集約に失敗した場合は警告を返さない．
func ConvertMarkdownDirToYAMLWithWarnings(mdDirPath, yamlFilePath string, recursive bool) ([]ValidationError, error) {
	items, err := AggregateMarkdownDir(mdDirPath, recursive)
	if err != nil {
		return nil, err
	}
	warnings := ValidateItems(items)
	return warnings, SaveYAMLData(items, yamlFilePath)
}
