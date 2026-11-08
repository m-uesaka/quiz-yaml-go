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

// YAMLファイルをCSVファイルに変換する．
// CSV出力には問題文、答え、原語表記、およびフォーマットされた正誤判定が含まれる．
// 組み込みのCSVテンプレート（templates/quiz_template.csv）で出力するため，
// ConvertWithBuiltinTemplate(yamlFilePath, csvFilePath, BuiltinCSV) と同じ結果になる．
func ConvertYAMLToCSV(yamlFilePath, csvFilePath string) error {
	return ConvertWithBuiltinTemplate(yamlFilePath, csvFilePath, BuiltinCSV)
}
