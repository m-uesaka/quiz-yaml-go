package quiz_yaml_converter

import (
	"bytes"
	"encoding/csv"
	"regexp"
	"strings"
	"unicode"
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

// numberPattern は，符号付きの整数・小数だけからなる値（例: -1，+5，-1.5，.5）に一致する．
// このような値は表計算ソフトで数値として扱われ，数式として実行されないため，EscapeCSVFormula の対象外にする．
var numberPattern = regexp.MustCompile(`^[+-]?([0-9]+(\.[0-9]*)?|\.[0-9]+)$`)

// 表計算ソフトで数式として解釈される値の先頭に「'」を付けて，文字列として扱わせる（CSV injection対策）．
// 先頭の空白（全角スペースなどUnicodeの空白を含む．タブ・CRは除く）を除いた最初の文字が
// 「=」「+」「-」「@」・タブ・CRのいずれかの場合に「'」を付ける．
// ただし，符号付きの数値だけからなる値（例: -1，+5，-1.5）は数式として実行されないため，そのまま返す．
// それ以外の値（空文字列を含む）もそのまま返す．全角の「＝」などは対象外．
// CSVの構文としてのエスケープは行わないため，CSVに出力する場合は CSVRowSafe / CSVFieldSafe を使う．
func EscapeCSVFormula(field string) string {
	trimmed := strings.TrimLeftFunc(field, func(r rune) bool {
		return r != '\t' && r != '\r' && unicode.IsSpace(r)
	})
	if trimmed == "" || numberPattern.MatchString(trimmed) {
		return field
	}
	switch trimmed[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + field
	}
	return field
}

// CSVRow と同じく引数を1行分のCSVレコードとしてフォーマットするが，
// 表計算ソフトで数式として解釈されないよう，次の2点を行う．
//   - 各フィールドに EscapeCSVFormula を適用する
//   - 「;」を含むフィールドは，CSVの規則ではクォートが不要でもダブルクォートで囲む
//     （「;」を区切り文字とする地域の表計算ソフトで，「a;=1+2」が別のセルに分かれて「=1+2」が実行されないようにするため）
//
// テンプレートからは {{csvRowSafe .Question .Answer}} のように使用する．
func CSVRowSafe(fields ...string) (string, error) {
	encoded := make([]string, len(fields))
	for i, field := range fields {
		// encoding/csv はフィールドごとに独立してクォートの要否を決めるため，1つずつ整形して「,」でつないでも CSVRow と同じ結果になる．
		s, err := CSVRow(EscapeCSVFormula(field))
		if err != nil {
			return "", err
		}
		// クォートされていないフィールドにはダブルクォートが含まれないため，そのまま囲めばよい．
		if strings.Contains(s, ";") && !strings.HasPrefix(s, `"`) {
			s = `"` + s + `"`
		}
		encoded[i] = s
	}
	return strings.Join(encoded, ","), nil
}

// CSVField と同じく1つの値をCSVのフィールドとしてエスケープするが，
// CSVRowSafe と同じく，表計算ソフトで数式として解釈されないようにする．
// テンプレートからは {{csvFieldSafe .Question}} のように使用する．
func CSVFieldSafe(field string) (string, error) {
	return CSVRowSafe(field)
}

// YAMLファイルをCSVファイルに変換する．
// CSV出力には問題文、答え、原語表記、およびフォーマットされた正誤判定が含まれる．
// 組み込みのCSVテンプレート（templates/quiz_template.csv）で出力するため，
// ConvertWithBuiltinTemplate(yamlFilePath, csvFilePath, BuiltinCSV) と同じ結果になる．
// 組み込みのCSVテンプレートは CSVRowSafe で出力するため，「=」「+」「-」「@」などで始まる値（数値のみの値を除く）の先頭には「'」が付く．
func ConvertYAMLToCSV(yamlFilePath, csvFilePath string) error {
	return ConvertWithBuiltinTemplate(yamlFilePath, csvFilePath, BuiltinCSV)
}
