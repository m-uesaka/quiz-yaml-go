package quiz_yaml_converter

import (
	"bytes"
	"encoding/csv"
	"strings"
)

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

// 表計算ソフトで数式として解釈される値の先頭に「'」を付けて，文字列として扱わせる（CSV injection対策）．
// 先頭の半角スペースを除いた最初の文字が「=」「+」「-」「@」・タブ・CRのいずれかの場合に「'」を付け，
// それ以外の値（空文字列を含む）はそのまま返す．
// CSVの構文としてのエスケープは行わないため，CSVに出力する場合は CSVRowSafe / CSVFieldSafe を使う．
func EscapeCSVFormula(field string) string {
	trimmed := strings.TrimLeft(field, " ")
	if trimmed == "" {
		return field
	}
	switch trimmed[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + field
	}
	return field
}

// CSVRow と同じく引数を1行分のCSVレコードとしてフォーマットするが，
// 各フィールドに EscapeCSVFormula を適用して，表計算ソフトで数式として解釈されないようにする．
// テンプレートからは {{csvRowSafe .Question .Answer}} のように使用する．
func CSVRowSafe(fields ...string) (string, error) {
	escaped := make([]string, len(fields))
	for i, field := range fields {
		escaped[i] = EscapeCSVFormula(field)
	}
	return CSVRow(escaped...)
}

// CSVField と同じく1つの値をCSVのフィールドとしてエスケープするが，
// EscapeCSVFormula を適用して，表計算ソフトで数式として解釈されないようにする．
// テンプレートからは {{csvFieldSafe .Question}} のように使用する．
func CSVFieldSafe(field string) (string, error) {
	return CSVRowSafe(field)
}

// YAMLファイルをCSVファイルに変換する．
// CSV出力には問題文、答え、原語表記、およびフォーマットされた正誤判定が含まれる．
// 組み込みのCSVテンプレート（templates/quiz_template.csv）で出力するため，
// ConvertWithBuiltinTemplate(yamlFilePath, csvFilePath, BuiltinCSV) と同じ結果になる．
// 組み込みのCSVテンプレートは CSVRowSafe で出力するため，「=」「+」「-」「@」などで始まる値の先頭には「'」が付く．
func ConvertYAMLToCSV(yamlFilePath, csvFilePath string) error {
	return ConvertWithBuiltinTemplate(yamlFilePath, csvFilePath, BuiltinCSV)
}
