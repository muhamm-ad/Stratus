package query

import (
	"testing"

	"github.com/muhamm-ad/stratus/internal/core"
)

func TestParseSearchFilterSort(t *testing.T) {
	q := Parse(`api p=aws state = running name+`)
	if len(q.Search) != 1 || q.Search[0] != "api" {
		t.Fatalf("search: %+v", q.Search)
	}
	if got := q.Values(FieldProvider); len(got) != 1 || got[0] != "aws" {
		t.Fatalf("provider: %+v", got)
	}
	if got := q.Values(FieldState); len(got) != 1 || got[0] != "running" {
		t.Fatalf("state: %+v", got)
	}
	if q.Sort.Field != FieldName || !q.Sort.Asc {
		t.Fatalf("sort: %+v", q.Sort)
	}
}

func TestParseAliasesAndSortForms(t *testing.T) {
	cases := []struct {
		in   string
		want Sort
	}{
		{"name+", Sort{FieldName, true}},
		{"name-", Sort{FieldName, false}},
		{"+provider", Sort{FieldProvider, true}},
		{"-region", Sort{FieldRegion, false}},
		{"s+", Sort{FieldState, true}},
		{"t:desc", Sort{FieldType, false}},
		{"name↗", Sort{FieldName, true}},
		{"name↘", Sort{FieldName, false}},
	}
	for _, tc := range cases {
		q := Parse(tc.in)
		if q.Sort != tc.want {
			t.Errorf("%q: got %+v want %+v", tc.in, q.Sort, tc.want)
		}
	}
}

func TestParseQuotedAndUnknownEquals(t *testing.T) {
	q := Parse(`"web api" foo=bar name="db 1"`)
	if len(q.Search) != 2 || q.Search[0] != "web api" || q.Search[1] != "foo=bar" {
		t.Fatalf("search: %+v", q.Search)
	}
	if got := q.Values(FieldName); len(got) != 1 || got[0] != "db 1" {
		t.Fatalf("name filter: %+v", got)
	}
}

func TestParseLastSortWins(t *testing.T) {
	q := Parse("name+ region-")
	if q.Sort.Field != FieldRegion || q.Sort.Asc {
		t.Fatalf("sort: %+v", q.Sort)
	}
}

func TestStringRoundTrip(t *testing.T) {
	in := `api provider=aws provider=gcp state=running name+`
	got := Parse(Parse(in).String())
	if got.String() != Parse(in).String() {
		t.Fatalf("round trip: %q vs %q", got.String(), Parse(in).String())
	}
}

func TestExactFilterVsSearch(t *testing.T) {
	vms := []core.VM{
		{Name: "web-1", Provider: "aws", Region: "us-east-1", Type: "t3.micro", State: core.StateRunning, ID: "i-1"},
		{Name: "web-2", Provider: "gcp", Region: "us-central1", Type: "e2-small", State: core.StateStopped, ID: "i-2"},
		{Name: "db", Provider: "aws", Region: "eu-west-1", Type: "t3.large", State: core.StateRunning, ID: "i-3"},
	}

	got := Apply(vms, Parse("web"))
	if len(got) != 2 {
		t.Fatalf("search web: %d", len(got))
	}

	got = Apply(vms, Parse("region=us"))
	if len(got) != 0 {
		t.Fatalf("region=us should be exact, got %d", len(got))
	}

	got = Apply(vms, Parse("region=us-east-1"))
	if len(got) != 1 || got[0].Name != "web-1" {
		t.Fatalf("region exact: %+v", got)
	}

	got = Apply(vms, Parse("p=aws state=running"))
	if len(got) != 2 {
		t.Fatalf("and filters: %d", len(got))
	}

	got = Apply(vms, Parse("provider=aws provider=gcp"))
	if len(got) != 3 {
		t.Fatalf("or same field: %d", len(got))
	}

	got = Apply(vms, Parse("web p=aws"))
	if len(got) != 1 || got[0].Name != "web-1" {
		t.Fatalf("search+filter: %+v", got)
	}
}

func TestSortAndTag(t *testing.T) {
	vms := []core.VM{
		{Name: "b", State: core.StateStopped, Tags: map[string]string{"env": "prod"}},
		{Name: "a", State: core.StateRunning, Tags: map[string]string{"env": "dev"}},
		{Name: "c", State: core.StateRunning, Tags: map[string]string{"env": "prod"}},
	}

	got := Apply(vms, Parse("name+"))
	if got[0].Name != "a" || got[1].Name != "b" || got[2].Name != "c" {
		t.Fatalf("name asc: %+v", names(got))
	}

	got = Apply(vms, Parse("name-"))
	if got[0].Name != "c" || got[2].Name != "a" {
		t.Fatalf("name desc: %+v", names(got))
	}

	got = Apply(vms, Parse("tag=env:prod"))
	if len(got) != 2 {
		t.Fatalf("tag: %d", len(got))
	}
}

func TestCycleHelpers(t *testing.T) {
	q := Parse("web")
	q = q.CycleFilter(FieldProvider, []string{"aws", "azure", "gcp", ""})
	if q.Values(FieldProvider)[0] != "aws" {
		t.Fatalf("first cycle: %s", q.String())
	}
	q = q.CycleFilter(FieldProvider, []string{"aws", "azure", "gcp", ""})
	if q.Values(FieldProvider)[0] != "azure" {
		t.Fatalf("second cycle: %s", q.String())
	}
	q = q.CycleSort()
	if q.Sort.Field != FieldName {
		t.Fatalf("sort cycle start: %+v", q.Sort)
	}
	q = q.ToggleSortDir()
	if q.Sort.Asc {
		t.Fatal("expected desc")
	}
	q = q.Clear()
	if !q.Empty() {
		t.Fatalf("clear: %s", q.String())
	}
}

func names(vms []core.VM) []string {
	out := make([]string, len(vms))
	for i, v := range vms {
		out[i] = v.Name
	}
	return out
}
