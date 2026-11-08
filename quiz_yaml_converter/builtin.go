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

// builtinTemplateSpec は組み込みテンプレート1つ分の設定．
type builtinTemplateSpec struct {
	file string // templates.FS 内のファイル名

	// escapeForHTMLOutput は，出力ファイルの拡張子がHTMLの場合に値をHTMLエスケープするかどうか．
	// ConvertToTemplate では出力ファイルの拡張子がHTMLならエスケープするが，CSVは従来の
	// ConvertYAMLToCSV と同じく，出力ファイルの拡張子に関係なくエスケープしない．
	// （テンプレート自体がHTMLの場合は，この設定に関係なく常にエスケープする．）
	escapeForHTMLOutput bool
}

// htmlMode は，このテンプレートで outputFilePath に出力する場合にHTMLモードを使うかどうかを返す．
func (s builtinTemplateSpec) htmlMode(outputFilePath string) bool {
	return isHTMLPath(s.file) || (s.escapeForHTMLOutput && isHTMLPath(outputFilePath))
}

// builtinTemplates は組み込みテンプレートの種類と，その設定の対応表．
var builtinTemplates = map[BuiltinTemplate]builtinTemplateSpec{
	BuiltinCSV:      {file: templates.CSV, escapeForHTMLOutput: false},
	BuiltinHTML:     {file: templates.HTML, escapeForHTMLOutput: true},
	BuiltinMarkdown: {file: templates.Markdown, escapeForHTMLOutput: true},
}

// lookupBuiltinTemplate は組み込みテンプレートの設定を返す．未対応の種類の場合はエラーを返す．
func lookupBuiltinTemplate(builtin BuiltinTemplate) (builtinTemplateSpec, error) {
	spec, ok := builtinTemplates[builtin]
	if !ok {
		return builtinTemplateSpec{}, fmt.Errorf("未対応の組み込みテンプレートです: %q", builtin)
	}
	return spec, nil
}

// ConvertWithBuiltinTemplate はYAMLファイルを組み込みテンプレートで変換し，出力ファイルに書き出す．
// ファイルシステム上のテンプレートを使う場合は Convert を使う．
// criteria の不正なキー以外のバリデーションは行わない．警告が必要な場合は ConvertWithBuiltinTemplateWithWarnings を使う．
func ConvertWithBuiltinTemplate(yamlFilePath, outputFilePath string, builtin BuiltinTemplate) error {
	data, spec, err := loadForBuiltinTemplate(yamlFilePath, builtin)
	if err != nil {
		return err
	}
	return convertToBuiltinTemplate(data, spec, outputFilePath)
}

// ConvertWithBuiltinTemplateWithWarnings は ConvertWithBuiltinTemplate と同じ変換を行い，
// 読み込んだデータを ValidateItems でバリデーションした結果を警告として返す．警告があっても変換は中断しない．
// 警告は読み込みに成功した場合に返し，その後の変換が失敗した場合も返す．
// criteria の不正なキーは警告ではなく，エラーになる．
func ConvertWithBuiltinTemplateWithWarnings(yamlFilePath, outputFilePath string, builtin BuiltinTemplate) ([]ValidationError, error) {
	data, spec, err := loadForBuiltinTemplate(yamlFilePath, builtin)
	if err != nil {
		return nil, err
	}
	warnings := ValidateItems(data)
	return warnings, convertToBuiltinTemplate(data, spec, outputFilePath)
}

// loadForBuiltinTemplate は組み込みテンプレートの種類が正しいことを確認してから，YAMLファイルを読み込む．
func loadForBuiltinTemplate(yamlFilePath string, builtin BuiltinTemplate) ([]QuizItem, builtinTemplateSpec, error) {
	spec, err := lookupBuiltinTemplate(builtin)
	if err != nil {
		return nil, builtinTemplateSpec{}, err
	}
	data, err := LoadYAMLData(yamlFilePath)
	return data, spec, err
}

// convertToBuiltinTemplate は問題データを組み込みテンプレートで変換し，出力ファイルに書き出す．
// HTMLモードを使うかどうかは builtinTemplateSpec.htmlMode で決める．
func convertToBuiltinTemplate(data []QuizItem, spec builtinTemplateSpec, outputFilePath string) error {
	content, err := fs.ReadFile(templates.FS, spec.file)
	if err != nil {
		return fmt.Errorf("組み込みテンプレートの読み込みに失敗しました: %w", err)
	}
	return executeTemplateToFile(string(content), spec.htmlMode(outputFilePath), data, outputFilePath)
}
