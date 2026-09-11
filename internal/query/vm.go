package query

import (
	"sort"
	"strings"

	"github.com/muhamm-ad/stratus/internal/core"
)

// Apply filters then sorts a copy of vms. The input slice is not modified.
func Apply(vms []core.VM, q Query) []core.VM {
	out := make([]core.VM, 0, len(vms))
	for _, vm := range vms {
		if q.Match(vm) {
			out = append(out, vm)
		}
	}
	SortVMs(out, q.Sort)
	return out
}

// Match reports whether vm satisfies search + filters.
func (q Query) Match(vm core.VM) bool {
	if !matchFilters(vm, q.Filters) {
		return false
	}
	if len(q.Search) == 0 {
		return true
	}
	hay := haystack(vm)
	for _, term := range q.Search {
		if !strings.Contains(hay, strings.ToLower(term)) {
			return false
		}
	}
	return true
}

func matchFilters(vm core.VM, filters []Filter) bool {
	if len(filters) == 0 {
		return true
	}
	byField := make(map[Field][]string, 8)
	for _, f := range filters {
		byField[f.Field] = append(byField[f.Field], f.Value)
	}
	for field, vals := range byField {
		ok := false
		for _, want := range vals {
			if fieldEquals(vm, field, want) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

func fieldEquals(vm core.VM, field Field, want string) bool {
	switch field {
	case FieldName:
		return strings.EqualFold(vm.Name, want)
	case FieldProvider:
		return strings.EqualFold(string(vm.Provider), want)
	case FieldRegion:
		return strings.EqualFold(string(vm.Region), want)
	case FieldType:
		return strings.EqualFold(string(vm.Type), want)
	case FieldState:
		return strings.EqualFold(string(vm.State), want)
	case FieldID:
		return strings.EqualFold(vm.ID, want)
	case FieldTag:
		return tagEquals(vm.Tags, want)
	default:
		return false
	}
}

func tagEquals(tags map[string]string, want string) bool {
	if len(tags) == 0 {
		return false
	}
	key, val, ok := strings.Cut(want, ":")
	if !ok {
		for k, v := range tags {
			if strings.EqualFold(k, want) || strings.EqualFold(v, want) {
				return true
			}
		}
		return false
	}
	for k, v := range tags {
		if strings.EqualFold(k, key) && strings.EqualFold(v, val) {
			return true
		}
	}
	return false
}

func haystack(vm core.VM) string {
	parts := []string{
		vm.Name,
		vm.ID,
		string(vm.Provider),
		string(vm.Region),
		string(vm.Type),
		string(vm.State),
	}
	return strings.ToLower(strings.Join(parts, "\x00"))
}

func fieldKey(vm core.VM, field Field) string {
	switch field {
	case FieldName:
		return vm.Name
	case FieldProvider:
		return string(vm.Provider)
	case FieldRegion:
		return string(vm.Region)
	case FieldType:
		return string(vm.Type)
	case FieldState:
		return string(vm.State)
	case FieldID:
		return vm.ID
	default:
		return ""
	}
}

// SortVMs sorts vms in place. A zero Sort is a no-op.
func SortVMs(vms []core.VM, s Sort) {
	if s.Field == "" || !sortable(s.Field) {
		return
	}
	sort.SliceStable(vms, func(i, j int) bool {
		a, b := fieldKey(vms[i], s.Field), fieldKey(vms[j], s.Field)
		if s.Asc {
			return a < b
		}
		return a > b
	})
}
