package broker

import "strings"

// fork-local: Method policy wraps upstream MatchService without changing its signature.

type MethodMatchStatus int

const (
	MethodMatchOK MethodMatchStatus = iota
	MethodMatchNoMatch
	MethodMatchDenied
)

func MatchServiceWithMethodPolicy(host string, targetPort int, path, method string, services []Service) (*Service, MatchScore, MethodMatchStatus) {
	matched, score := MatchService(host, targetPort, path, services)
	if matched == nil {
		return nil, score, MethodMatchNoMatch
	}
	if MethodAllowed(matched.Methods, method) {
		return matched, score, MethodMatchOK
	}
	return matched, score, MethodMatchDenied
}

func MethodAllowed(methods []string, method string) bool {
	normalized, err := NormalizeMethodList(methods)
	if err != nil {
		return false
	}
	if len(normalized) == 0 {
		return true
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	for _, allowed := range normalized {
		if method == allowed {
			return true
		}
	}
	return false
}
