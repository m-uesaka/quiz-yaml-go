package quiz_yaml_converter

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	texttemplate "text/template"
	"time"
)

// HTMLとして扱う拡張子（大文字・小文字は区別しない）．
var htmlExtensions = map[string]bool{
	".html":  true,
	".htm":   true,
	".xhtml": true,
}

// パスの拡張子がHTMLのものかどうかを返す．
func isHTMLPath(path string) bool {
	return htmlExtensions[strings.ToLower(filepath.Ext(path))]
}

// html/templateとtext/templateのどちらのテンプレートも満たすインターフェース．
type templateExecutor interface {
	Execute(w io.Writer, data any) error
}

// テンプレートに提供する関数のうち，エスケープモードに依存しないもの．
func baseTemplateFuncs() map[string]any {
	return map[string]any{
		"formatCriteria": (*Criteria).Format,
		"quoteAll":       formatCriteriaSection,
		"csvRow":         CSVRow,
		"csvField":       CSVField,
		"addQuotes":      AddQuotesIfNeeded,
		"join":           strings.Join,
		"upper":          strings.ToUpper,
		"lower":          strings.ToLower,
		"replace":        strings.ReplaceAll,
		"add": func(a, b int) int {
			return a + b
		},
		"now": func() string {
			return time.Now().Format("2006年01月02日 15:04:05")
		},
	}
}

// テンプレートを解析する．
// HTMLモードではhtml/templateを使用して値を自動的にエスケープし，
// それ以外ではtext/templateを使用して値をそのまま出力する．
// どちらのモードでも safeHTML と escapeHTML が使用でき，同じテンプレートが両方で動作する．
func parseTemplate(content string, htmlMode bool) (templateExecutor, error) {
	funcs := baseTemplateFuncs()
	if htmlMode {
		// エスケープ済みであることを示すため template.HTML を返し，二重エスケープを防ぐ．
		funcs["escapeHTML"] = func(s string) htmltemplate.HTML {
			return htmltemplate.HTML(htmltemplate.HTMLEscapeString(s))
		}
		funcs["safeHTML"] = func(s string) htmltemplate.HTML {
			return htmltemplate.HTML(s)
		}
		return htmltemplate.New("quiz").Funcs(funcs).Parse(content)
	}
	funcs["escapeHTML"] = texttemplate.HTMLEscapeString
	funcs["safeHTML"] = func(s string) string { return s }
	return texttemplate.New("quiz").Funcs(funcs).Parse(content)
}

// 問題データとテンプレートファイルから出力ファイルを生成する．
// テンプレートファイルか出力ファイルの拡張子が .html / .htm / .xhtml の場合は
// html/templateを使用して値を自動的にエスケープし，それ以外はtext/templateを使用する．
func ConvertToTemplate(data []QuizItem, templateFilePath, outputFilePath string) error {
	templateContent, err := os.ReadFile(templateFilePath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	htmlMode := isHTMLPath(templateFilePath) || isHTMLPath(outputFilePath)
	return executeTemplateToFile(string(templateContent), htmlMode, data, outputFilePath)
}

// テンプレートの内容contentを解析・実行し，結果を出力ファイルに書き込む．
// ファイルシステム上のテンプレートと組み込みテンプレートの両方で使う共通の出力処理．
func executeTemplateToFile(content string, htmlMode bool, data []QuizItem, outputFilePath string) error {
	tmpl, err := parseTemplate(content, htmlMode)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// テンプレートの実行が途中で失敗した場合（html/templateのエスケープエラーを含む）に
	// 既存の出力ファイルを壊さないよう，一度メモリ上に出力し，実行に成功した場合のみファイルに書き込む．
	// os.WriteFile は書き込みとクローズのどちらのエラーも返すため，書き込みの失敗も呼び出し元に伝わる．
	// ただし os.WriteFile は既存のファイルを空にしてから書き込むため，書き込みの途中で失敗した場合は
	// 既存の出力ファイルは残らず，途中まで書き込まれたファイルになる．
	// 権限は os.Create と同じく 0666 から umask を引いたもの（新しくファイルを作る場合のみ）．
	var buf bytes.Buffer
	templateData := TemplateData{Items: data}
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	if err := os.WriteFile(outputFilePath, buf.Bytes(), 0o666); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}
