package query

import (
	"strings"
	"unicode"
)

type sortSuffix struct {
	tok string
	asc bool
}

// Longer tokens first so :desc wins over a lone suffix character.
var sortSuffixes = []sortSuffix{
	{":desc", false},
	{":asc", true},
	{"↘", false},
	{"↗", true},
	{"↓", false},
	{"↑", true},
	{"+", true},
	{"-", false},
	{"^", true},
}

var sortPrefixes = []sortSuffix{
	{"↘", false},
	{"↗", true},
	{"↓", false},
	{"↑", true},
	{"+", true},
	{"-", false},
}

// Parse turns a query string into a Query. Invalid bits become search terms.
func Parse(s string) Query {
	var q Query
	for _, tok := range mergeEq(tokenize(s)) {
		if tok == "" {
			continue
		}
		if sort, ok := parseSortToken(tok); ok {
			q.Sort = sort
			continue
		}
		if field, value, ok := splitFilter(tok); ok {
			q.Filters = append(q.Filters, Filter{Field: field, Value: value})
			continue
		}
		q.Search = append(q.Search, tok)
	}
	return q
}

func tokenize(s string) []string {
	var (
		tokens []string
		buf    strings.Builder
		quote  bool
	)
	flush := func() {
		if buf.Len() == 0 {
			return
		}
		tokens = append(tokens, buf.String())
		buf.Reset()
	}
	for _, r := range s {
		switch {
		case r == '"':
			quote = !quote
		case unicode.IsSpace(r) && !quote:
			flush()
		default:
			buf.WriteRune(r)
		}
	}
	flush()
	return tokens
}

// mergeEq turns `provider`, `=`, `aws` (and similar splits) into one token.
func mergeEq(tokens []string) []string {
	out := make([]string, 0, len(tokens))
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if i+2 < len(tokens) && isFieldName(tok) && tokens[i+1] == "=" {
			out = append(out, tok+"="+tokens[i+2])
			i += 2
			continue
		}
		if i+1 < len(tokens) && isFieldName(tok) && strings.HasPrefix(tokens[i+1], "=") {
			out = append(out, tok+tokens[i+1])
			i++
			continue
		}
		if i+1 < len(tokens) && strings.HasSuffix(tok, "=") && isFieldName(strings.TrimSuffix(tok, "=")) {
			out = append(out, tok+tokens[i+1])
			i++
			continue
		}
		out = append(out, tok)
	}
	return out
}

func isFieldName(s string) bool {
	_, ok := CanonicalField(s)
	return ok
}

func splitFilter(tok string) (Field, string, bool) {
	key, val, ok := strings.Cut(tok, "=")
	if !ok {
		return "", "", false
	}
	field, ok := CanonicalField(key)
	if !ok {
		return "", "", false
	}
	return field, strings.TrimSpace(val), true
}

func parseSortToken(tok string) (Sort, bool) {
	for _, suf := range sortSuffixes {
		if !strings.HasSuffix(tok, suf.tok) {
			continue
		}
		name := strings.TrimSuffix(tok, suf.tok)
		if f, ok := CanonicalField(name); ok && sortable(f) {
			return Sort{Field: f, Asc: suf.asc}, true
		}
	}
	for _, pre := range sortPrefixes {
		if !strings.HasPrefix(tok, pre.tok) {
			continue
		}
		name := strings.TrimPrefix(tok, pre.tok)
		if f, ok := CanonicalField(name); ok && sortable(f) {
			return Sort{Field: f, Asc: pre.asc}, true
		}
	}
	return Sort{}, false
}

func sortable(f Field) bool {
	return f != FieldTag
}
