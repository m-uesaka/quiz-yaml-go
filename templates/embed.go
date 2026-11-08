// Package templates は，組み込みテンプレート（quiz_template.csv / .html / .md）を
// go:embed でバイナリに埋め込んで提供する．
// 埋め込んだテンプレートは実行時のカレントディレクトリに関係なく読み込める．
package templates

import "embed"

// 組み込みテンプレートのファイル名．FS から読み込むときの名前として使う．
const (
	CSV      = "quiz_template.csv"  // CSV出力用テンプレート
	HTML     = "quiz_template.html" // HTML出力用テンプレート
	Markdown = "quiz_template.md"   // Markdown出力用テンプレート
)

// FS は組み込みテンプレートを埋め込んだファイルシステム．
//
//go:embed quiz_template.csv quiz_template.html quiz_template.md
var FS embed.FS
