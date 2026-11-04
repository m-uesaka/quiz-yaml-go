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

// ConvertMarkdownDirToYAML はMarkdownディレクトリを1つのYAMLファイルに集約する
// エントリーポイント．recursiveがtrueの場合はサブディレクトリも再帰的に辿る．
func ConvertMarkdownDirToYAML(mdDirPath, yamlFilePath string, recursive bool) error {
	items, err := AggregateMarkdownDir(mdDirPath, recursive)
	if err != nil {
		return err
	}
	return SaveYAMLData(items, yamlFilePath)
}
