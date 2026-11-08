// Package quizconv は，YAMLフォーマットの問題データをテンプレート処理，もしくはCSV形式に変換するためのパッケージです．
//
//	import "github.com/m-uesaka/quiz-yaml-go/v2/quizconv"
//
// v1.x までは quiz_yaml_converter というパッケージ名（import "github.com/m-uesaka/quiz-yaml-go/quiz_yaml_converter"）でした．
//
// このパッケージが提供する機能は以下の通りです：
//   - YAMLファイルからの問題データの読み込み
//   - 日本語テキストを適切に処理したCSV形式への変換
//   - Goのテンプレートを使用したカスタムフォーマットへの変換
//   - 日本語クイズフォーマット用の組み込みテンプレート関数
package quizconv
