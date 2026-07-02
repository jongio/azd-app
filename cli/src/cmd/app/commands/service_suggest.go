package commands

import (
	"fmt"
	"strings"
)

// serviceSuggestMaxDistance caps how far apart (in case-insensitive edit
// distance) a requested name and a defined service name may be before the name
// stops being offered as a "did you mean" suggestion. It is deliberately small
// so that only genuine typos are suggested, not unrelated names.
const serviceSuggestMaxDistance = 3

// resolveServiceName validates a requested service name against the list of
// available service names. It returns the matched name on an exact
// (case-sensitive) match, otherwise a formatted "not found" error that lists
// the available services and, when a close match exists, a single "Did you
// mean X?" suggestion.
//
// The returned error is a plain error value so callers can surface it as text
// or embed its message in a structured (JSON) error field without extra
// formatting.
func resolveServiceName(requested string, available []string) (string, error) {
	for _, name := range available {
		if name == requested {
			return name, nil
		}
	}
	return "", serviceNotFoundError(requested, available)
}

// serviceNotFoundError builds a consistent error for an unknown service name.
// It always lists the available services and appends a single suggestion when
// one is close enough (case-insensitive, small edit distance).
func serviceNotFoundError(requested string, available []string) error {
	if len(available) == 0 {
		return fmt.Errorf("service %q not found. No services are defined in azure.yaml", requested)
	}

	msg := fmt.Sprintf("service %q not found. Available services: %s",
		requested, strings.Join(available, ", "))

	if suggestion := suggestServiceName(requested, available); suggestion != "" {
		msg += fmt.Sprintf(". Did you mean %q?", suggestion)
	}

	return fmt.Errorf("%s", msg)
}

// suggestServiceName returns the single closest available service name to the
// requested name, or an empty string when nothing is close. Matching is
// case-insensitive, but the returned name keeps its original casing.
func suggestServiceName(requested string, available []string) string {
	reqLower := strings.ToLower(strings.TrimSpace(requested))
	if reqLower == "" {
		return ""
	}

	best := ""
	bestDist := -1
	for _, name := range available {
		dist := levenshtein(reqLower, strings.ToLower(name))
		if bestDist == -1 || dist < bestDist {
			bestDist = dist
			best = name
		}
	}

	if best == "" {
		return ""
	}

	// Scale the allowed distance with the requested length so that short names
	// require a closer match, and cap it so unrelated names are never offered.
	threshold := len(reqLower)/2 + 1
	if threshold > serviceSuggestMaxDistance {
		threshold = serviceSuggestMaxDistance
	}
	if bestDist > threshold {
		return ""
	}

	return best
}

// levenshtein computes the Levenshtein edit distance between two strings using
// the standard two-row dynamic programming approach.
func levenshtein(a, b string) int {
	ar := []rune(a)
	br := []rune(b)
	if len(ar) == 0 {
		return len(br)
	}
	if len(br) == 0 {
		return len(ar)
	}

	prev := make([]int, len(br)+1)
	curr := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(ar); i++ {
		curr[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			curr[j] = min(
				curr[j-1]+1,    // insertion
				prev[j]+1,      // deletion
				prev[j-1]+cost, // substitution
			)
		}
		prev, curr = curr, prev
	}

	return prev[len(br)]
}
