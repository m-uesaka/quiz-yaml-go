package quiz_yaml_converter

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeTempMarkdown(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	return p
}

func TestParseMarkdownFile_FullySpecified(t *testing.T) {
	content := `---
title: 自作問題-ハヤカワ・ポケット・ミステリ
date: 2026-06-12
tags:
    - ガツオ
---
## Question

ミッキー・スピレインの『大いなる殺人』を第一冊として1953年に刊行された，
早川書房の翻訳ミステリーシリーズは何でしょう？

## Answer

ハヤカワ・ポケット・ミステリ

## Spell

Hayakawa Pocket Mystery

## Criteria

### OK

- 「ハヤカワ・ミステリ」

### NG

- 「ハヤカワ・ミステリ文庫」（別レーベル）

### Close

## Comment
`
	dir := t.TempDir()
	path := writeTempMarkdown(t, dir, "test.md", content)

	item, err := ParseMarkdownFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Answer != "ハヤカワ・ポケット・ミステリ" {
		t.Errorf("Answer = %q", item.Answer)
	}
	if item.Spell != "Hayakawa Pocket Mystery" {
		t.Errorf("Spell = %q", item.Spell)
	}
	if len(item.Tags) != 1 || item.Tags[0] != "ガツオ" {
		t.Errorf("Tags = %v", item.Tags)
	}
	if got := item.Criteria["ok"]; len(got) != 1 || got[0] != "「ハヤカワ・ミステリ」" {
		t.Errorf("Criteria.ok = %v", got)
	}
	if got := item.Criteria["ng"]; len(got) != 1 || got[0] != "「ハヤカワ・ミステリ文庫」（別レーベル）" {
		t.Errorf("Criteria.ng = %v", got)
	}
	if _, exists := item.Criteria["repeat"]; exists {
		t.Errorf("Criteria.repeat should be absent when ### Close is empty")
	}
	if item.Comments != nil {
		t.Errorf("Comments = %v, want nil", item.Comments)
	}
}

func TestParseMarkdownFile_EmptyOptionalSections(t *testing.T) {
	content := `---
title: 自作問題-Vessel
date: 2026-03-20
---
## Question

デッキと階段が蜂の巣状に外周を取り囲む建築作品は何でしょう？

## Answer

Vessel

## Spell

## Criteria

### OK

### NG

### Close

## Comment
`
	dir := t.TempDir()
	path := writeTempMarkdown(t, dir, "vessel.md", content)

	item, err := ParseMarkdownFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Spell != "" {
		t.Errorf("Spell = %q, want empty", item.Spell)
	}
	if item.Comments != nil {
		t.Errorf("Comments = %v, want nil", item.Comments)
	}
	if item.Criteria != nil {
		t.Errorf("Criteria = %v, want nil", item.Criteria)
	}
	if item.Tags != nil {
		t.Errorf("Tags = %v, want nil (frontmatter has no tags key)", item.Tags)
	}
}

func TestParseMarkdownFile_MultilineQuestionWithoutBlankLine(t *testing.T) {
	content := `---
title: 自作問題-ジェプツンタンパ1世
date: 2026-06-12
tags:
    - ガツオ
---
## Question

1974年にモンゴルで化石が発見された恐竜に献名されているのは誰でしょう？
【通常】ジュンガルの首長ガルダンと対立したときには宗教指導者は誰でしょう？

## Answer

ジェプツンタンパ1世

## Spell

Jebtsundamba Khutuktu

## Criteria

### OK

- ウンドル・ゲゲン・ザナザバル（Öndör Gegeen Zanabazar）
- ジュニャーナヴァジュラ（Jñānavajra）

### NG

### Close

## Comment
`
	dir := t.TempDir()
	path := writeTempMarkdown(t, dir, "test.md", content)

	item, err := ParseMarkdownFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantQuestion := "1974年にモンゴルで化石が発見された恐竜に献名されているのは誰でしょう？\n" +
		"【通常】ジュンガルの首長ガルダンと対立したときには宗教指導者は誰でしょう？"
	if item.Question != wantQuestion {
		t.Errorf("Question = %q, want %q", item.Question, wantQuestion)
	}
}

func TestParseMarkdownFile_CommentWithMultipleLinesAndMarkdownLink(t *testing.T) {
	content := `---
title: 自作問題-Vessel
date: 2026-03-20
---
## Question

質問文

## Answer

Vessel

## Spell

## Criteria

### OK

### NG

### Close

## Comment

[4人が命を落としたNYの巨大アートが3年ぶりに公開再開 | ARTnews JAPAN](https://artnewsjapan.com/article/2792)
これは補足コメントの2行目である．

別の段落のコメント．
`
	dir := t.TempDir()
	path := writeTempMarkdown(t, dir, "test.md", content)

	item, err := ParseMarkdownFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantFirst := "[4人が命を落としたNYの巨大アートが3年ぶりに公開再開 | ARTnews JAPAN](https://artnewsjapan.com/article/2792)\n" +
		"これは補足コメントの2行目である．"
	wantSecond := "別の段落のコメント．"
	if len(item.Comments) != 2 {
		t.Fatalf("Comments = %v, want 2 elements", item.Comments)
	}
	if item.Comments[0] != wantFirst {
		t.Errorf("Comments[0] = %q, want %q", item.Comments[0], wantFirst)
	}
	if item.Comments[1] != wantSecond {
		t.Errorf("Comments[1] = %q, want %q", item.Comments[1], wantSecond)
	}
}

func TestParseMarkdownFile_CloseMapsToRepeat(t *testing.T) {
	content := `---
title: 自作問題-テスト
date: 2026-01-01
---
## Question

問題文

## Answer

答え

## Spell

## Criteria

### OK

- 別解1

### NG

### Close

- もう一度1
- もう一度2

## Comment
`
	dir := t.TempDir()
	path := writeTempMarkdown(t, dir, "test.md", content)

	item, err := ParseMarkdownFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"もう一度1", "もう一度2"}
	if got := item.Criteria["repeat"]; !reflect.DeepEqual(got, want) {
		t.Errorf("Criteria.repeat = %v, want %v", got, want)
	}
	if _, exists := item.Criteria["ng"]; exists {
		t.Errorf("Criteria.ng should be absent when ### NG is empty")
	}
}

func TestAggregateMarkdownDir_SortedAndAggregated(t *testing.T) {
	dir := t.TempDir()
	fixture := func(answer string) string {
		return "---\ntitle: 自作問題-" + answer + "\ndate: 2026-01-01\n---\n" +
			"## Question\n\n問題文\n\n## Answer\n\n" + answer + "\n\n## Spell\n\n## Criteria\n\n### OK\n\n### NG\n\n### Close\n\n## Comment\n"
	}
	writeTempMarkdown(t, dir, "c.md", fixture("C"))
	writeTempMarkdown(t, dir, "a.md", fixture("A"))
	writeTempMarkdown(t, dir, "b.md", fixture("B"))

	items, err := AggregateMarkdownDir(dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}
	want := []string{"A", "B", "C"}
	for i, w := range want {
		if items[i].Answer != w {
			t.Errorf("items[%d].Answer = %q, want %q", i, items[i].Answer, w)
		}
	}
}

func TestAggregateMarkdownDir_Recursive(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}
	fixture := func(answer string) string {
		return "---\ntitle: 自作問題-" + answer + "\ndate: 2026-01-01\n---\n" +
			"## Question\n\n問題文\n\n## Answer\n\n" + answer + "\n\n## Spell\n\n## Criteria\n\n### OK\n\n### NG\n\n### Close\n\n## Comment\n"
	}
	writeTempMarkdown(t, dir, "top.md", fixture("TOP"))
	writeTempMarkdown(t, sub, "nested.md", fixture("NESTED"))

	nonRecursive, err := AggregateMarkdownDir(dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nonRecursive) != 1 {
		t.Fatalf("non-recursive len(items) = %d, want 1", len(nonRecursive))
	}

	recursive, err := AggregateMarkdownDir(dir, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recursive) != 2 {
		t.Fatalf("recursive len(items) = %d, want 2", len(recursive))
	}
}

func TestAggregateMarkdownDir_PartialFailureReportsFileName(t *testing.T) {
	dir := t.TempDir()
	goodContent := "---\ntitle: 自作問題-OK\ndate: 2026-01-01\n---\n" +
		"## Question\n\n問題文\n\n## Answer\n\nOK\n\n## Spell\n\n## Criteria\n\n### OK\n\n### NG\n\n### Close\n\n## Comment\n"
	writeTempMarkdown(t, dir, "good.md", goodContent)
	brokenPath := writeTempMarkdown(t, dir, "broken.md", "no frontmatter here")

	items, err := AggregateMarkdownDir(dir, false)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), brokenPath) {
		t.Errorf("error message %q does not contain broken file path %q", err.Error(), brokenPath)
	}
	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1 (only the successfully parsed file)", len(items))
	}
}

func TestSaveYAMLData_RoundTrip(t *testing.T) {
	original := []QuizItem{
		{
			Question: "問題文1\n複数行",
			Answer:   "答え1",
			Spell:    "Answer1",
			Tags:     []string{"タグ1", "タグ2"},
			Comments: []string{"コメント1", "コメント2\n2行目"},
			Criteria: map[string][]string{
				"ok":     {"別解1"},
				"ng":     {"誤答1"},
				"repeat": {"もう一度1"},
			},
		},
		{
			Question: "問題文2",
			Answer:   "答え2",
		},
	}

	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "out.yaml")

	if err := SaveYAMLData(original, yamlPath); err != nil {
		t.Fatalf("SaveYAMLData failed: %v", err)
	}

	loaded, err := LoadYAMLData(yamlPath)
	if err != nil {
		t.Fatalf("LoadYAMLData failed: %v", err)
	}

	if !reflect.DeepEqual(original, loaded) {
		t.Errorf("round-trip mismatch:\noriginal = %+v\nloaded   = %+v", original, loaded)
	}
}
