package quiz_yaml_converter

import (
	"fmt"
	"os"
	"strings"
)

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
