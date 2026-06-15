package broker

import "testing"

func TestMatchServiceWithMethodPolicyAllowsListedMethod(t *testing.T) {
	services := []Service{{
		Name:    "calendar-read",
		Host:    "www.googleapis.com",
		Path:    "/calendar/v3/calendars/*/events*",
		Methods: []string{"GET"},
	}}

	svc, _, status := MatchServiceWithMethodPolicy("www.googleapis.com", 443, "/calendar/v3/calendars/primary/events", "GET", services)
	if status != MethodMatchOK {
		t.Fatalf("status = %v, want MethodMatchOK", status)
	}
	if svc == nil || svc.Name != "calendar-read" {
		t.Fatalf("service = %+v, want calendar-read", svc)
	}
}

func TestMatchServiceWithMethodPolicyDeniesUnlistedMethod(t *testing.T) {
	services := []Service{{
		Name:    "calendar-read",
		Host:    "www.googleapis.com",
		Path:    "/calendar/v3/calendars/*/events*",
		Methods: []string{"GET"},
	}}

	svc, _, status := MatchServiceWithMethodPolicy("www.googleapis.com", 443, "/calendar/v3/calendars/primary/events", "POST", services)
	if status != MethodMatchDenied {
		t.Fatalf("status = %v, want MethodMatchDenied", status)
	}
	if svc == nil || svc.Name != "calendar-read" {
		t.Fatalf("service = %+v, want calendar-read", svc)
	}
}

func TestMatchServiceWithMethodPolicyOmittedMethodsAllowAny(t *testing.T) {
	services := []Service{{
		Name: "any-method",
		Host: "api.example.com",
	}}

	_, _, status := MatchServiceWithMethodPolicy("api.example.com", 443, "/", "PATCH", services)
	if status != MethodMatchOK {
		t.Fatalf("status = %v, want MethodMatchOK", status)
	}
}

func TestMatchServiceWithMethodPolicyNoFallthroughToBroaderRule(t *testing.T) {
	services := []Service{
		{
			Name:    "gmail-read",
			Host:    "gmail.googleapis.com",
			Path:    "/gmail/v1/users/me/messages*",
			Methods: []string{"GET"},
		},
		{
			Name: "gmail-broad",
			Host: "gmail.googleapis.com",
			Path: "/*",
		},
	}

	svc, _, status := MatchServiceWithMethodPolicy("gmail.googleapis.com", 443, "/gmail/v1/users/me/messages", "POST", services)
	if status != MethodMatchDenied {
		t.Fatalf("status = %v, want MethodMatchDenied", status)
	}
	if svc == nil || svc.Name != "gmail-read" {
		t.Fatalf("service = %+v, want gmail-read", svc)
	}
}

func TestMethodAllowedInvalidStoredPolicyFailsClosed(t *testing.T) {
	if MethodAllowed([]string{"GET", "GET"}, "GET") {
		t.Fatal("duplicate stored method policy should fail closed")
	}
}
