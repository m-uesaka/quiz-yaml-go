package quiz_yaml_converter

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

// ValidationError はバリデーションで見つかった問題1件を表す構造体．
// error インターフェースを実装しており，Error() は「問題 1: 答え (answer) が空です」のような
// 人が読むためのメッセージを返す．
type ValidationError struct {
	Item    int    // 問題番号（1始まり）．ファイル全体に関する問題の場合は0
	Field   string // 問題のあるフィールド（例: "answer", "criteria.ok[1]"）．特定のフィールドに関係しない場合は空文字
	Message string // 問題番号を含まないメッセージ（例: "答え (answer) が空です"）
	Source  string // 問題の元になったファイルのパス（Markdownの集約時に設定）．不明な場合は空文字
}

// Error は問題番号を含めたメッセージを返す．
// Itemが1以上の場合は「問題 <Item>: <Message>」，0の場合はMessageをそのまま返す．
// Sourceが設定されている場合は「問題 <Item> (<Source>): <Message>」のように元のファイルを示す．
func (e ValidationError) Error() string {
	if e.Item <= 0 {
		if e.Source != "" {
			return fmt.Sprintf("%s: %s", e.Source, e.Message)
		}
		return e.Message
	}
	if e.Source != "" {
		return fmt.Sprintf("問題 %d (%s): %s", e.Item, e.Source, e.Message)
	}
	return fmt.Sprintf("問題 %d: %s", e.Item, e.Message)
}

// String は Error() と同じ文字列を返す．
func (e ValidationError) String() string {
	return e.Error()
}

// ValidationResult はバリデーション結果を表す構造体
type ValidationResult struct {
	IsValid bool              // バリデーションが成功したかどうか
	Errors  []string          // エラーメッセージのリスト（Detailsの各要素の Error() と同じ内容）
	Items   int               // 読み込まれたアイテム数
	Details []ValidationError // 構造化されたエラーのリスト（Errorsと同じ順序）
}

// addError はエラーを1件追加し，IsValidをfalseにする．
func (r *ValidationResult) addError(e ValidationError) {
	r.IsValid = false
	r.Details = append(r.Details, e)
	r.Errors = append(r.Errors, e.Error())
}

// ValidateYAMLFile はYAMLファイルの構造と内容をバリデーションする
func ValidateYAMLFile(yamlFilePath string) ValidationResult {
	result := ValidationResult{
		IsValid: true,
		Errors:  []string{},
		Items:   0,
	}

	// YAMLデータの読み込み
	// criteria の不正なキーは，ほかのエラーと一緒に報告するため ValidateItems で確認する．
	data, err := decodeYAMLFile(yamlFilePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			result.addError(ValidationError{Message: fmt.Sprintf("ファイルが存在しません: %s", yamlFilePath)})
		} else {
			result.addError(ValidationError{Message: fmt.Sprintf("YAMLファイルの読み込みエラー: %v", err)})
		}
		return result
	}

	result.Items = len(data)

	// 各アイテムのバリデーション（データが空であることの確認を含む）
	for _, e := range ValidateItems(data) {
		result.addError(e)
	}

	return result
}

// ValidateItems はクイズデータの各問題をバリデーションし，見つかった問題を返す．
// 問題が無い場合はnilを返す．ファイルを読み込まずにデータだけを検査するため，
// 変換時やMarkdownの集約時にも使える．
//
// 次の内容を確認する．
//   - クイズデータが1件以上ある（空の場合は Item が0のエラーを1件返す）
//   - 問題文 (question) と答え (answer) が空でない（空白のみも空とみなす）
//   - criteria の ok / ng / repeat，comments，tags に空の要素が無い
//   - criteria に ok / ng / repeat 以外のキーが無い（LoadYAMLData で読み込んだデータでは常に無い）
func ValidateItems(items []QuizItem) []ValidationError {
	if len(items) == 0 {
		return []ValidationError{{Message: "YAMLファイルにクイズデータが含まれていません"}}
	}

	var errs []ValidationError
	for i, item := range items {
		errs = append(errs, validateQuizItem(item, i+1)...)
	}
	return errs
}

// validateQuizItem は個々のクイズアイテムをバリデーションする．indexは1始まりの問題番号．
func validateQuizItem(item QuizItem, index int) []ValidationError {
	var errs []ValidationError

	// 必須フィールドのチェック
	if strings.TrimSpace(item.Question) == "" {
		errs = append(errs, ValidationError{Item: index, Field: "question", Message: "問題文 (question) が空です"})
	}

	if strings.TrimSpace(item.Answer) == "" {
		errs = append(errs, ValidationError{Item: index, Field: "answer", Message: "答え (answer) が空です"})
	}

	// criteriaフィールドのバリデーション
	for _, msg := range item.Criteria.unknownKeyErrors("") {
		errs = append(errs, ValidationError{Item: index, Field: "criteria", Message: msg})
	}
	c := item.criteria()
	errs = append(errs, checkNonEmpty(index, "criteria.ok", c.OK)...)
	errs = append(errs, checkNonEmpty(index, "criteria.ng", c.NG)...)
	errs = append(errs, checkNonEmpty(index, "criteria.repeat", c.Repeat)...)

	// comments / tags フィールドのバリデーション
	errs = append(errs, checkNonEmpty(index, "comments", item.Comments)...)
	errs = append(errs, checkNonEmpty(index, "tags", item.Tags)...)

	return errs
}

// checkNonEmpty はvaluesのうち空（空白のみを含む）の要素ごとに，
// 「<field>[<要素番号>] が空です」というエラーを返す．要素番号は0始まり．
// indexは1始まりの問題番号．
func checkNonEmpty(index int, field string, values []string) []ValidationError {
	var errs []ValidationError
	for j, v := range values {
		if strings.TrimSpace(v) == "" {
			name := fmt.Sprintf("%s[%d]", field, j)
			errs = append(errs, ValidationError{Item: index, Field: name, Message: name + " が空です"})
		}
	}
	return errs
}
