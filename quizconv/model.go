package quizconv

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

// 出力ファイルの形式（CSV形式またはテンプレート形式）を表す型．
// DetectOutputFormat が返す．値は形式を区別するためのもので，出力される内容や拡張子ではない．
type OutputFormat string

// 出力形式
const (
	FormatCSV      OutputFormat = "csv"      // CSV形式
	FormatTemplate OutputFormat = "template" // テンプレート形式
)
