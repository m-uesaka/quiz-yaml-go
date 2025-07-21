# Quiz Questions

{{range $index, $item := .Items}}
## Question {{add $index 1}}

{{with .Comments}}**Comments:** {{join . ", "}}{{end}}

**Q:** {{.Question}}

**Answer:** {{.Answer}}

{{if .Criteria}}
**Criteria:** {{formatCriteria .Criteria}}
{{end}}

---

{{end}}
