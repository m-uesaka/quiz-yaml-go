# Quiz Questions

{{range $index, $item := .Items}}
## Question {{add $index 1}}

{{with .Comments}}**Comments:** {{join . ", " | escapeHTML | nl2br}}{{end}}

**Q:** {{escapeHTML .Question | nl2br}}

**Answer:** {{escapeHTML .Answer | nl2br}}

{{if .HasCriteria}}
**Criteria:** {{formatCriteria .Criteria | escapeHTML | nl2br}}
{{end}}

---

{{end}}
