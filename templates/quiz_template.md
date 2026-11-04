# Quiz Questions

{{range $index, $item := .Items}}
## Question {{add $index 1}}

{{with .Comments}}**Comments:** {{join . ", " | escapeHTML}}{{end}}

**Q:** {{escapeHTML .Question}}

**Answer:** {{escapeHTML .Answer}}

{{if .Criteria}}
**Criteria:** {{formatCriteria .Criteria | escapeHTML}}
{{end}}

---

{{end}}
