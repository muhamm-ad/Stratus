// Package query is the inventory search / filter / sort language.
//
// Every UI (query bar, keyboard shortcuts, command palette, later the GUI)
// must parse and apply queries here so matchers do not drift. Spec:
// docs/inventory-query.md
package query

import "strings"

// Field is a canonical inventory column (or tag/id).
type Field string

const (
	FieldName     Field = "name"
	FieldProvider Field = "provider"
	FieldRegion   Field = "region"
	FieldType     Field = "type"
	FieldState    Field = "state"
	FieldID       Field = "id"
	FieldTag      Field = "tag"
)

// SortFields is the cycle order for the `o` shortcut (tag is not sortable).
var SortFields = []Field{
	FieldName, FieldProvider, FieldRegion, FieldType, FieldState,
}

var fieldOrder = []Field{
	FieldName, FieldProvider, FieldRegion, FieldType, FieldState, FieldID, FieldTag,
}

var aliases = map[string]Field{
	"name": FieldName, "n": FieldName,
	"provider": FieldProvider, "p": FieldProvider,
	"region": FieldRegion, "r": FieldRegion,
	"type": FieldType, "t": FieldType,
	"state": FieldState, "s": FieldState,
	"id":  FieldID,
	"tag": FieldTag,
}

// CanonicalField maps a name or alias to a field.
func CanonicalField(name string) (Field, bool) {
	f, ok := aliases[strings.ToLower(strings.TrimSpace(name))]
	return f, ok
}

// Filter is one exact-match clause.
type Filter struct {
	Field Field
	Value string
}

// Sort is a single column + direction. A zero Field means "keep load order".
type Sort struct {
	Field Field
	Asc   bool
}

// Query is the parsed form of the inventory query string.
type Query struct {
	Search  []string
	Filters []Filter
	Sort    Sort
}

// Empty reports whether the query does nothing.
func (q Query) Empty() bool {
	return len(q.Search) == 0 && len(q.Filters) == 0 && q.Sort.Field == ""
}

// Values returns every filter value for field, in query order.
func (q Query) Values(field Field) []string {
	var out []string
	for _, f := range q.Filters {
		if f.Field == field {
			out = append(out, f.Value)
		}
	}
	return out
}

// SetFilter replaces all clauses for field. Empty values removes the field.
func (q Query) SetFilter(field Field, values ...string) Query {
	q.Filters = cloneFiltersWithout(q.Filters, field)
	for _, v := range values {
		if v == "" {
			continue
		}
		q.Filters = append(q.Filters, Filter{Field: field, Value: v})
	}
	return q
}

// ClearFilter drops every clause for field.
func (q Query) ClearFilter(field Field) Query {
	q.Filters = cloneFiltersWithout(q.Filters, field)
	return q
}

// Clear returns an empty query.
func (q Query) Clear() Query { return Query{} }

// CycleFilter walks values (last entry may be "" to mean "off").
func (q Query) CycleFilter(field Field, values []string) Query {
	if len(values) == 0 {
		return q.ClearFilter(field)
	}
	cur := ""
	if vs := q.Values(field); len(vs) == 1 {
		cur = vs[0]
	}
	idx := -1
	for i, v := range values {
		if strings.EqualFold(v, cur) {
			idx = i
			break
		}
	}
	next := values[0]
	if idx >= 0 {
		next = values[(idx+1)%len(values)]
	}
	if next == "" {
		return q.ClearFilter(field)
	}
	return q.SetFilter(field, next)
}

// CycleSort walks SortFields, then off. Direction is kept when moving columns.
func (q Query) CycleSort() Query {
	if q.Sort.Field == "" {
		q.Sort = Sort{Field: FieldName, Asc: true}
		return q
	}
	for i, f := range SortFields {
		if q.Sort.Field == f {
			if i+1 >= len(SortFields) {
				q.Sort = Sort{}
				return q
			}
			q.Sort.Field = SortFields[i+1]
			return q
		}
	}
	q.Sort = Sort{Field: FieldName, Asc: q.Sort.Asc}
	return q
}

// ToggleSortDir flips ascending/descending. No-op when unsorted.
func (q Query) ToggleSortDir() Query {
	if q.Sort.Field == "" {
		return q
	}
	q.Sort.Asc = !q.Sort.Asc
	return q
}

// SetSort sets the sort column and direction. Empty field clears sort.
func (q Query) SetSort(field Field, asc bool) Query {
	if field == "" {
		q.Sort = Sort{}
		return q
	}
	q.Sort = Sort{Field: field, Asc: asc}
	return q
}

func cloneFiltersWithout(in []Filter, field Field) []Filter {
	if len(in) == 0 {
		return nil
	}
	out := make([]Filter, 0, len(in))
	for _, f := range in {
		if f.Field != field {
			out = append(out, f)
		}
	}
	return out
}

// String rebuilds a canonical query (aliases expanded, sort as + / -).
func (q Query) String() string {
	var parts []string
	for _, t := range q.Search {
		parts = append(parts, quoteToken(t))
	}
	for _, field := range fieldOrder {
		for _, f := range q.Filters {
			if f.Field != field {
				continue
			}
			parts = append(parts, string(field)+"="+quoteToken(f.Value))
		}
	}
	if q.Sort.Field != "" {
		dir := "+"
		if !q.Sort.Asc {
			dir = "-"
		}
		parts = append(parts, string(q.Sort.Field)+dir)
	}
	return strings.Join(parts, " ")
}

func quoteToken(s string) string {
	if s == "" || strings.ContainsAny(s, " \t=") || looksSpecial(s) {
		return `"` + strings.ReplaceAll(s, `"`, "") + `"`
	}
	return s
}

func looksSpecial(s string) bool {
	if _, _, ok := splitFilter(s); ok {
		return true
	}
	_, ok := parseSortToken(s)
	return ok
}
