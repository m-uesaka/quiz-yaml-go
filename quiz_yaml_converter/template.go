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
// now はテンプレート関数 now が返す時刻の取得元（通常は time.Now）．
func baseTemplateFuncs(now func() time.Time) map[string]any {
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
		// 時刻は now が返したタイムゾーンのまま（通常はローカル時刻で）出力する．
		"now": func() string {
			return now().Format("2006年01月02日 15:04:05")
		},
	}
}

// テンプレートを解析する．
// HTMLモードではhtml/templateを使用して値を自動的にエスケープし，
// それ以外ではtext/templateを使用して値をそのまま出力する．
// どちらのモードでも safeHTML と escapeHTML が使用でき，同じテンプレートが両方で動作する．
// テンプレート関数 now は現在時刻（time.Now）を返す．
func parseTemplate(content string, htmlMode bool) (templateExecutor, error) {
	return parseTemplateWithClock(content, htmlMode, time.Now)
}

// parseTemplateWithClock は parseTemplate と同じくテンプレートを解析する．
// テンプレート関数 now は，引数の now が返す時刻を出力する．テストで時刻を固定するために使う．
func parseTemplateWithClock(content string, htmlMode bool, now func() time.Time) (templateExecutor, error) {
	funcs := baseTemplateFuncs(now)
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
	tmpl, err := parseTemplate(string(templateContent), htmlMode)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// テンプレートの実行が途中で失敗した場合（html/templateのエスケープエラーを含む）に
	// 既存の出力ファイルを壊さないよう，一度メモリ上に出力してから，成功した場合のみファイルに書き込む．
	var buf bytes.Buffer
	templateData := TemplateData{Items: data}
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	if err := os.WriteFile(outputFilePath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}
