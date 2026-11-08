package quiz_yaml_converter

import (
	"fmt"
	"io/fs"

	"github.com/m-uesaka/quiz-yaml-go/templates"
)

// BuiltinTemplate は組み込みテンプレートの種類を表す．
// 組み込みテンプレートはバイナリに埋め込まれているため，実行時のカレントディレクトリに関係なく使用できる．
type BuiltinTemplate string

// 組み込みテンプレートの種類
const (
	BuiltinCSV      BuiltinTemplate = "csv"      // CSV（templates/quiz_template.csv）
	BuiltinHTML     BuiltinTemplate = "html"     // HTML（templates/quiz_template.html）
	BuiltinMarkdown BuiltinTemplate = "markdown" // Markdown（templates/quiz_template.md）
)

// builtinTemplateFiles は組み込みテンプレートの種類と，templates.FS 内のファイル名の対応表．
var builtinTemplateFiles = map[BuiltinTemplate]string{
	BuiltinCSV:      templates.CSV,
	BuiltinHTML:     templates.HTML,
	BuiltinMarkdown: templates.Markdown,
}

// ConvertWithBuiltinTemplate はYAMLファイルを組み込みテンプレートで変換し，出力ファイルに書き出す．
// ファイルシステム上のテンプレートを使う場合は Convert を使う．
// criteria の不正なキー以外のバリデーションは行わない．警告が必要な場合は ConvertWithBuiltinTemplateWithWarnings を使う．
func ConvertWithBuiltinTemplate(yamlFilePath, outputFilePath string, builtin BuiltinTemplate) error {
	data, err := loadForBuiltinTemplate(yamlFilePath, builtin)
	if err != nil {
		return err
	}
	return convertToBuiltinTemplate(data, builtin, outputFilePath)
}

// ConvertWithBuiltinTemplateWithWarnings は ConvertWithBuiltinTemplate と同じ変換を行い，
// 読み込んだデータを ValidateItems でバリデーションした結果を警告として返す．警告があっても変換は中断しない．
// 警告は読み込みに成功した場合に返し，その後の変換が失敗した場合も返す．
// criteria の不正なキーは警告ではなく，エラーになる．
func ConvertWithBuiltinTemplateWithWarnings(yamlFilePath, outputFilePath string, builtin BuiltinTemplate) ([]ValidationError, error) {
	data, err := loadForBuiltinTemplate(yamlFilePath, builtin)
	if err != nil {
		return nil, err
	}
	warnings := ValidateItems(data)
	return warnings, convertToBuiltinTemplate(data, builtin, outputFilePath)
}

// loadForBuiltinTemplate は組み込みテンプレートの種類が正しいことを確認してから，YAMLファイルを読み込む．
func loadForBuiltinTemplate(yamlFilePath string, builtin BuiltinTemplate) ([]QuizItem, error) {
	if _, ok := builtinTemplateFiles[builtin]; !ok {
		return nil, fmt.Errorf("unsupported builtin template: %q", builtin)
	}
	return LoadYAMLData(yamlFilePath)
}

// convertToBuiltinTemplate は問題データを組み込みテンプレートで変換し，出力ファイルに書き出す．
// HTMLモードの判定は ConvertToTemplate と同じく，テンプレートか出力ファイルの拡張子で行う．
// ただしCSVは，出力ファイルの拡張子に関係なく常にエスケープせずに出力する．
func convertToBuiltinTemplate(data []QuizItem, builtin BuiltinTemplate, outputFilePath string) error {
	name, ok := builtinTemplateFiles[builtin]
	if !ok {
		return fmt.Errorf("unsupported builtin template: %q", builtin)
	}
	content, err := fs.ReadFile(templates.FS, name)
	if err != nil {
		return fmt.Errorf("failed to read builtin template: %w", err)
	}
	htmlMode := isHTMLPath(name) || (builtin != BuiltinCSV && isHTMLPath(outputFilePath))
	return executeTemplateToFile(string(content), htmlMode, data, outputFilePath)
}
