package proposal

import (
	"reflect"
	"testing"

	"github.com/Infisical/agent-vault/internal/broker"
)

func TestValidateRejectsInvalidServiceMethods(t *testing.T) {
	auth := broker.Auth{Type: "bearer", Token: "EXAMPLE_TOKEN"}
	services := []Service{{
		Action:  ActionSet,
		Name:    "example",
		Host:    "api.example.com",
		Auth:    &auth,
		Methods: []string{"GET", "GET"},
	}}

	if err := Validate(services, nil); err == nil {
		t.Fatal("Validate expected duplicate method error")
	}
}

func TestMergeServicesPreservesAndClearsMethods(t *testing.T) {
	auth := broker.Auth{Type: "bearer", Token: "EXAMPLE_TOKEN"}
	existing := []broker.Service{{
		Name:    "example",
		Host:    "api.example.com",
		Auth:    auth,
		Methods: []string{"GET"},
	}}

	merged, warnings := MergeServices(existing, []Service{{
		Action:  ActionSet,
		Name:    "example",
		Host:    "api.example.com",
		Auth:    &auth,
		Methods: []string{"post"},
	}})
	if len(warnings) > 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if want := []string{"POST"}; !reflect.DeepEqual(merged[0].Methods, want) {
		t.Fatalf("methods = %v, want %v", merged[0].Methods, want)
	}

	merged, _ = MergeServices(merged, []Service{{
		Action: ActionSet,
		Name:   "example",
		Host:   "api.example.com",
		Auth:   &auth,
	}})
	if len(merged[0].Methods) != 0 {
		t.Fatalf("methods = %v, want unrestricted", merged[0].Methods)
	}
}
