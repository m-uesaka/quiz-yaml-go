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
// criteria の不正なキー以外のバリデーションは行わない．警告が必要な場合は ConvertWithWarnings を使う．
func Convert(yamlFilePath, outputFilePath, templateFilePath string) error {
	data, err := loadForConvert(yamlFilePath, outputFilePath, templateFilePath)
	if err != nil {
		return err
	}
	return writeOutput(data, outputFilePath, templateFilePath)
}

// ConvertWithWarnings は Convert と同じ変換を行い，読み込んだデータを ValidateItems で
// バリデーションした結果を警告として返す．警告があっても変換は中断しない．
// 警告は読み込みに成功した場合に返し，その後の変換が失敗した場合も返す．
// criteria の不正なキーは警告ではなく，従来どおりエラーになる．
func ConvertWithWarnings(yamlFilePath, outputFilePath, templateFilePath string) ([]ValidationError, error) {
	data, err := loadForConvert(yamlFilePath, outputFilePath, templateFilePath)
	if err != nil {
		return nil, err
	}
	warnings := ValidateItems(data)
	return warnings, writeOutput(data, outputFilePath, templateFilePath)
}

// loadForConvert は出力フォーマットに必要な引数がそろっていることを確認してから，
// 変換するYAMLファイルを読み込む．
func loadForConvert(yamlFilePath, outputFilePath, templateFilePath string) ([]QuizItem, error) {
	if DetectOutputFormat(outputFilePath, templateFilePath) == FormatTemplate && templateFilePath == "" {
		return nil, fmt.Errorf("template file is required for non-CSV output")
	}
	return LoadYAMLData(yamlFilePath)
}

// writeOutput はデータを出力ファイルのフォーマットに合わせて書き出す．
func writeOutput(data []QuizItem, outputFilePath, templateFilePath string) error {
	if DetectOutputFormat(outputFilePath, templateFilePath) == FormatCSV {
		return writeCSV(data, outputFilePath)
	}
	return ConvertToTemplate(data, templateFilePath, outputFilePath)
}

// ConvertMarkdownDirToYAML はMarkdownディレクトリを1つのYAMLファイルに集約する
// エントリーポイント．recursiveがtrueの場合はサブディレクトリも再帰的に辿る．
// バリデーションは行わない．警告が必要な場合は ConvertMarkdownDirToYAMLWithWarnings を使う．
func ConvertMarkdownDirToYAML(mdDirPath, yamlFilePath string, recursive bool) error {
	items, err := AggregateMarkdownDir(mdDirPath, recursive)
	if err != nil {
		return err
	}
	return SaveYAMLData(items, yamlFilePath)
}

// ConvertMarkdownDirToYAMLWithWarnings は ConvertMarkdownDirToYAML と同じ集約を行い，
// 集約したデータを ValidateItems でバリデーションした結果を警告として返す．
// 警告の問題番号は出力するYAMLファイルでの順番（1始まり）を表し，Source には
// 元になったMarkdownファイルのパスを設定する．
// 警告があってもYAMLファイルは書き出す．一部のファイルの変換に失敗した場合は
// YAMLファイルを書き出さないが，変換できたファイルの警告はエラーと一緒に返す．
func ConvertMarkdownDirToYAMLWithWarnings(mdDirPath, yamlFilePath string, recursive bool) ([]ValidationError, error) {
	items, sources, err := aggregateMarkdownDir(mdDirPath, recursive)
	if err != nil {
		if len(items) == 0 {
			return nil, err
		}
		return validateMarkdownItems(items, sources), err
	}
	warnings := validateMarkdownItems(items, sources)
	return warnings, SaveYAMLData(items, yamlFilePath)
}

// validateMarkdownItems は集約したデータを ValidateItems でバリデーションし，
// 各警告の Source に元になったMarkdownファイルのパスを設定する．
// sources[i] は items[i] のファイルのパス．
func validateMarkdownItems(items []QuizItem, sources []string) []ValidationError {
	warnings := ValidateItems(items)
	for i := range warnings {
		if n := warnings[i].Item; n >= 1 && n <= len(sources) {
			warnings[i].Source = sources[n-1]
		}
	}
	return warnings
}
