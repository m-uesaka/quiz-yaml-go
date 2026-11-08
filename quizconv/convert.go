package quizconv

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
// CSV形式は組み込みのCSVテンプレート（templates/quiz_template.csv）で出力する．
// criteria の不正なキー以外のバリデーションは行わない．警告が必要な場合は ConvertWithWarnings を使う．
func Convert(yamlFilePath, outputFilePath, templateFilePath string) error {
	data, format, err := loadForConvert(yamlFilePath, outputFilePath, templateFilePath)
	if err != nil {
		return err
	}
	return writeOutput(data, format, outputFilePath, templateFilePath)
}

// ConvertWithWarnings は Convert と同じ変換を行い，読み込んだデータを ValidateItems で
// バリデーションした結果を警告として返す．警告があっても変換は中断しない．
// 警告は読み込みに成功した場合に返し，その後の変換が失敗した場合も返す．
// criteria の不正なキーは警告ではなく，従来どおりエラーになる．
func ConvertWithWarnings(yamlFilePath, outputFilePath, templateFilePath string) ([]ValidationError, error) {
	data, format, err := loadForConvert(yamlFilePath, outputFilePath, templateFilePath)
	if err != nil {
		return nil, err
	}
	warnings := ValidateItems(data)
	return warnings, writeOutput(data, format, outputFilePath, templateFilePath)
}

// loadForConvert は出力フォーマットを検出し，そのフォーマットに必要な引数が
// そろっていることを確認してから，変換するYAMLファイルを読み込む．
// 検出したフォーマットは writeOutput にそのまま渡す．
func loadForConvert(yamlFilePath, outputFilePath, templateFilePath string) ([]QuizItem, OutputFormat, error) {
	format := DetectOutputFormat(outputFilePath, templateFilePath)
	if format == FormatTemplate && templateFilePath == "" {
		return nil, format, fmt.Errorf("CSV以外の形式で出力するにはテンプレートファイルが必要です")
	}
	data, err := LoadYAMLData(yamlFilePath)
	return data, format, err
}

// writeOutput はデータを format に合わせて書き出す．
// DetectOutputFormat に新しいフォーマットを追加して，ここへの追加を忘れた場合に
// 別のフォーマットで書き出してしまわないよう，未対応のフォーマットはエラーにする．
func writeOutput(data []QuizItem, format OutputFormat, outputFilePath, templateFilePath string) error {
	switch format {
	case FormatCSV:
		return convertToBuiltinTemplate(data, builtinTemplates[BuiltinCSV], outputFilePath)
	case FormatTemplate:
		return ConvertToTemplate(data, templateFilePath, outputFilePath)
	default:
		return fmt.Errorf("未対応の出力形式です: %q", format)
	}
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
