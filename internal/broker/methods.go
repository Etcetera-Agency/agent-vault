package broker

import (
	"fmt"
	"strings"
)

// fork-local: Method policy helpers keep Agent Vault fork edits isolated from upstream matching code.

// NormalizeMethods validates and normalizes Service.Methods in place.
func NormalizeMethods(svc *Service) error {
	methods, err := NormalizeMethodList(svc.Methods)
	if err != nil {
		return err
	}
	svc.Methods = methods
	return nil
}

// NormalizeMethodList uppercases valid methods and collapses ["*"] to unrestricted.
func NormalizeMethodList(methods []string) ([]string, error) {
	if len(methods) == 0 {
		return nil, nil
	}

	normalized := make([]string, 0, len(methods))
	seen := make(map[string]bool, len(methods))
	for _, method := range methods {
		method = strings.ToUpper(strings.TrimSpace(method))
		if method == "" {
			return nil, fmt.Errorf("method must not be empty")
		}
		if method == "*" {
			if len(methods) == 1 {
				return nil, nil
			}
			return nil, fmt.Errorf("wildcard \"*\" is only valid as the sole method")
		}
		if !isHTTPToken(method) {
			return nil, fmt.Errorf("invalid HTTP method %q", method)
		}
		if seen[method] {
			return nil, fmt.Errorf("duplicate HTTP method %q", method)
		}
		seen[method] = true
		normalized = append(normalized, method)
	}
	return normalized, nil
}

// DisplayMethods renders empty storage as the canonical any-method token.
func DisplayMethods(methods []string) []string {
	normalized, err := NormalizeMethodList(methods)
	if err != nil || len(normalized) == 0 {
		return []string{"*"}
	}
	return normalized
}

func isHTTPToken(method string) bool {
	for i := 0; i < len(method); i++ {
		if !isTChar(method[i]) {
			return false
		}
	}
	return true
}

func isTChar(c byte) bool {
	if c >= 'A' && c <= 'Z' {
		return true
	}
	if c >= '0' && c <= '9' {
		return true
	}
	return strings.ContainsRune("!#$%&'*+-.^_`|~", rune(c))
}
