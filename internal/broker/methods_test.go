package broker

import (
	"reflect"
	"testing"
)

func TestNormalizeMethodListValid(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "empty means unrestricted"},
		{name: "uppercase", in: []string{"GET", "HEAD"}, want: []string{"GET", "HEAD"}},
		{name: "lowercase normalized", in: []string{"get", "post"}, want: []string{"GET", "POST"}},
		{name: "trimmed", in: []string{" GET ", " head "}, want: []string{"GET", "HEAD"}},
		{name: "extension token", in: []string{"m-search"}, want: []string{"M-SEARCH"}},
		{name: "wildcard means unrestricted", in: []string{"*"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeMethodList(tc.in)
			if err != nil {
				t.Fatalf("NormalizeMethodList(%v) error: %v", tc.in, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("NormalizeMethodList(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalizeMethodListRejectsInvalid(t *testing.T) {
	tests := [][]string{
		{""},
		{" "},
		{"GET", "GET"},
		{"*", "GET"},
		{"GET", "*"},
		{"GET /foo"},
	}

	for _, tc := range tests {
		if _, err := NormalizeMethodList(tc); err == nil {
			t.Fatalf("NormalizeMethodList(%v) expected error", tc)
		}
	}
}

func TestValidateNormalizesMethods(t *testing.T) {
	cfg := Config{
		Vault: "default",
		Services: []Service{{
			Name:    "example",
			Host:    "api.example.com",
			Auth:    Auth{Type: "bearer", Token: "EXAMPLE_TOKEN"},
			Methods: []string{"get", "head"},
		}},
	}

	if err := Validate(&cfg); err != nil {
		t.Fatalf("Validate error: %v", err)
	}
	want := []string{"GET", "HEAD"}
	if !reflect.DeepEqual(cfg.Services[0].Methods, want) {
		t.Fatalf("methods = %v, want %v", cfg.Services[0].Methods, want)
	}
}
