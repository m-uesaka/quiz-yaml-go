// YAMLフォーマットの問題データをテンプレート処理，もしくはCSV形式に変換するためのパッケージです．
// このパッケージが提供する機能は以下の通りです：
//   - YAMLファイルからの問題データの読み込み
//   - 日本語テキストを適切に処理したCSV形式への変換
//   - Goのテンプレートを使用したカスタムフォーマットへの変換
//   - 日本語クイズフォーマット用の組み込みテンプレート関数
package quiz_yaml_converter

import (
	"bytes"
	"encoding/csv"
	"fmt"
	htmltemplate "html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	texttemplate "text/template"
	"time"

	"gopkg.in/yaml.v3"
)

// 1問ごとのエントリを表す構造体
// 問題文、答え、原語表記、コメント、および判定基準を含む。
type QuizItem struct {
	Question string    `yaml:"question"`           // 問題文
	Answer   string    `yaml:"answer"`             // 答え
	Spell    string    `yaml:"spell"`              // 原語表記（英語表記）
	Tags     []string  `yaml:"tags,omitempty"`     // タグ
	Comments []string  `yaml:"comments,omitempty"` // コメント
	Criteria *Criteria `yaml:"criteria,omitempty"` // 判定基準（判定基準が無い場合はnil）
}

// 1問分の判定基準を表す構造体．
// YAMLの criteria オブジェクトに対応する．ok / ng / repeat 以外のキーは読み込み時に記録され，
// LoadYAMLData ではエラー，ValidateYAMLFile ではバリデーションエラーとして報告される．
type Criteria struct {
	OK     []string `yaml:"ok,omitempty"`     // 別解（正解として扱う答え）
	NG     []string `yaml:"ng,omitempty"`     // 誤答として扱う答え
	Repeat []string `yaml:"repeat,omitempty"` // もう一度回答を求める答え

	unknownKeys []unknownCriteriaKey // YAMLに書かれていた ok / ng / repeat 以外のキー
}

// criteria に書かれていた不正なキーと，その行番号．
type unknownCriteriaKey struct {
	name string
	line int
}

// YAMLの criteria オブジェクトを読み込む．
// ok / ng / repeat はそのまま読み込み，それ以外のキーは不正なキーとして記録する．
// criteria 以外のフィールドの未定義のキーは従来どおり無視するため，Decoder.KnownFields は使わない．
func (c *Criteria) UnmarshalYAML(node *yaml.Node) error {
	type plain Criteria // UnmarshalYAMLを持たない型に変換し，再帰呼び出しを防ぐ
	if err := node.Decode((*plain)(c)); err != nil {
		return err
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		if key.Tag == "!!merge" { // <<: *anchor によるマージは不正なキーとして扱わない
			continue
		}
		switch key.Value {
		case "ok", "ng", "repeat":
			continue
		}
		c.unknownKeys = append(c.unknownKeys, unknownCriteriaKey{name: key.Value, line: key.Line})
	}
	return nil
}

// 不正なキーごとのエラーメッセージを返す．prefixは「問題 1: 」のような問題の位置を表す文字列．
func (c *Criteria) unknownKeyErrors(prefix string) []string {
	if c == nil {
		return nil
	}
	var errs []string
	for _, key := range c.unknownKeys {
		errs = append(errs, fmt.Sprintf("%s不正なcriteriaキー: '%s'（%d行目） (使用可能: ok, ng, repeat)", prefix, key.name, key.line))
	}
	return errs
}

// 判定基準が1件も含まれていない（nilを含む）かどうかを返す．
func (c *Criteria) IsEmpty() bool {
	return c == nil || (len(c.OK) == 0 && len(c.NG) == 0 && len(c.Repeat) == 0)
}

// 判定基準を値で返す．Criteriaがnilの場合はゼロ値（すべてのリストがnil）を返す．
func (q QuizItem) criteria() Criteria {
	if q.Criteria == nil {
		return Criteria{}
	}
	return *q.Criteria
}

// 判定基準が1件以上含まれているかどうかを返す．
// Criteriaがnilの場合も，criteria: {} のように中身が空の場合もfalseを返す．
// テンプレートからは {{if .HasCriteria}} のように使用する．
func (q QuizItem) HasCriteria() bool {
	return !q.Criteria.IsEmpty()
}

// 判定基準のうち別解（ok）のリストを返す．
// テンプレートからは {{.OK}} でアクセスできる．Criteriaがnilの場合はnilを返す．
func (q QuizItem) OK() []string {
	return q.criteria().OK
}

// 判定基準のうち誤答（ng）のリストを返す．
// テンプレートからは {{.NG}} でアクセスできる．Criteriaがnilの場合はnilを返す．
func (q QuizItem) NG() []string {
	return q.criteria().NG
}

// 判定基準のうちもう一度（repeat）のリストを返す．
// テンプレートからは {{.Repeat}} でアクセスできる．Criteriaがnilの場合はnilを返す．
func (q QuizItem) Repeat() []string {
	return q.criteria().Repeat
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
//
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
// 空のリストは出力に含めない．Criteriaがnilの場合は空文字を返す．
// テンプレートからは {{formatCriteria .Criteria}} のように使用する．
func (c *Criteria) Format() string {
	if c == nil {
		return ""
	}

	var parts []string
	if len(c.OK) > 0 {
		parts = append(parts, formatCriteriaSection(c.OK, ""))
	}
	if len(c.NG) > 0 {
		parts = append(parts, formatCriteriaSection(c.NG, "は誤答"))
	}
	if len(c.Repeat) > 0 {
		parts = append(parts, formatCriteriaSection(c.Repeat, "はもう一度"))
	}

	return strings.Join(parts, "／")
}

// 引数を1行分のCSVレコードとしてフォーマットする（末尾の改行は含まない）．
// カンマ・ダブルクォート・改行などを含むフィールドはencoding/csvと同じ規則でクォートされる．
// テンプレートからは {{csvRow .Question .Answer}} のように使用する．
func CSVRow(fields ...string) (string, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(fields); err != nil {
		return "", err
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

// 1つの値をCSVのフィールドとしてエスケープする．
// テンプレートからは {{csvField .Question}} のように使用する．
func CSVField(field string) (string, error) {
	return CSVRow(field)
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
// criteria に ok / ng / repeat 以外のキーがある場合はエラーになる．
// criteria 以外のフィールドの未定義のキーは無視する．
func LoadYAMLData(yamlFilePath string) ([]QuizItem, error) {
	data, err := decodeYAMLFile(yamlFilePath)
	if err != nil {
		return nil, err
	}

	var errs []string
	for i, item := range data {
		errs = append(errs, item.Criteria.unknownKeyErrors(fmt.Sprintf("問題 %d: ", i+1))...)
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid criteria keys:\n%s", strings.Join(errs, "\n"))
	}

	return data, nil
}

// YAMLファイルを読み込んでQuizItemのスライスに変換する．
// criteria の不正なキーはエラーにせず，Criteria に記録したまま返す．
func decodeYAMLFile(yamlFilePath string) ([]QuizItem, error) {
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
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return data, nil
}

// ValidationResult はバリデーション結果を表す構造体
type ValidationResult struct {
	IsValid bool     // バリデーションが成功したかどうか
	Errors  []string // エラーメッセージのリスト
	Items   int      // 読み込まれたアイテム数
}

// ValidateYAMLFile はYAMLファイルの構造と内容をバリデーションする
func ValidateYAMLFile(yamlFilePath string) ValidationResult {
	result := ValidationResult{
		IsValid: true,
		Errors:  []string{},
		Items:   0,
	}

	// ファイルの存在確認
	if _, err := os.Stat(yamlFilePath); os.IsNotExist(err) {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("ファイルが存在しません: %s", yamlFilePath))
		return result
	}

	// YAMLデータの読み込み
	// criteria の不正なキーは，ほかのエラーと一緒に報告するため validateQuizItem で確認する．
	data, err := decodeYAMLFile(yamlFilePath)
	if err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("YAMLファイルの読み込みエラー: %v", err))
		return result
	}

	result.Items = len(data)

	// 各アイテムのバリデーション
	for i, item := range data {
		itemErrors := validateQuizItem(item, i+1)
		if len(itemErrors) > 0 {
			result.IsValid = false
			result.Errors = append(result.Errors, itemErrors...)
		}
	}

	// 配列が空でないことを確認
	if len(data) == 0 {
		result.IsValid = false
		result.Errors = append(result.Errors, "YAMLファイルにクイズデータが含まれていません")
	}

	return result
}

// validateQuizItem は個々のクイズアイテムをバリデーションする
func validateQuizItem(item QuizItem, index int) []string {
	var errs []string
	prefix := fmt.Sprintf("問題 %d: ", index)

	// 必須フィールドのチェック
	if strings.TrimSpace(item.Question) == "" {
		errs = append(errs, prefix+"問題文 (question) が空です")
	}

	if strings.TrimSpace(item.Answer) == "" {
		errs = append(errs, prefix+"答え (answer) が空です")
	}

	// criteriaフィールドのバリデーション
	errs = append(errs, item.Criteria.unknownKeyErrors(prefix)...)
	if item.Criteria != nil {
		sections := []struct {
			key   string
			items []string
		}{
			{"ok", item.Criteria.OK},
			{"ng", item.Criteria.NG},
			{"repeat", item.Criteria.Repeat},
		}
		for _, section := range sections {
			for j, answer := range section.items {
				if strings.TrimSpace(answer) == "" {
					errs = append(errs, fmt.Sprintf("%scriteria.%s[%d] が空です", prefix, section.key, j))
				}
			}
		}
	}

	// commentsフィールドのバリデーション
	for j, comment := range item.Comments {
		if strings.TrimSpace(comment) == "" {
			errs = append(errs, fmt.Sprintf("%scomments[%d] が空です", prefix, j))
		}
	}

	// tagsフィールドのバリデーション
	for j, tag := range item.Tags {
		if strings.TrimSpace(tag) == "" {
			errs = append(errs, fmt.Sprintf("%stags[%d] が空です", prefix, j))
		}
	}

	return errs
}

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
		err = writer.Write([]string{
			item.Question,
			item.Answer,
			item.Spell,
			item.Criteria.Format(),
		})
		if err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// QuizItemのスライスをYAMLファイルとして書き出す．
func SaveYAMLData(items []QuizItem, yamlFilePath string) error {
	out, err := yaml.Marshal(items)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	if err := os.WriteFile(yamlFilePath, out, 0o644); err != nil {
		return fmt.Errorf("failed to write YAML file: %w", err)
	}
	return nil
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
