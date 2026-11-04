package quiz_yaml_converter

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
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
