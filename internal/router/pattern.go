package router

import (
	"fmt"
	"strings"
)

// segment represents a single path segment in a URL pattern.
type segment struct {
	value   string // The segment value (e.g., "users", "id")
	isParam bool   // True if this is a parameter (e.g., :id), false if static
}

// pattern represents a parsed URL path pattern with segments and metadata.
type pattern struct {
	segments []segment // Parsed segments from the path
	raw      string    // Original path string
}

// parsePattern parses a URL path pattern into segments.
// Parameters are denoted with a leading colon (e.g., :id).
// Returns an error if the pattern is malformed.
func parsePattern(path string) (*pattern, error) {
	// Split by "/" and skip the leading empty string from the leading "/"
	parts := strings.Split(path, "/")
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}

	// Handle root path: "/" becomes [""] after split and skip
	if len(parts) == 0 {
		parts = []string{""}
	}

	segments := make([]segment, 0, len(parts))
	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			// This is a parameter
			paramName := strings.TrimPrefix(part, ":")
			segments = append(segments, segment{
				value:   paramName,
				isParam: true,
			})
		} else {
			// This is a static segment
			segments = append(segments, segment{
				value:   part,
				isParam: false,
			})
		}
	}

	return &pattern{
		segments: segments,
		raw:      path,
	}, nil
}

// validate checks if the pattern is valid.
// Returns an error if:
// - There are duplicate parameter names
// - There are wildcard segments (*)
// - There are empty parameter names
func (p *pattern) validate() error {
	seenParams := make(map[string]bool)

	for _, seg := range p.segments {
		if seg.isParam {
			// Check for empty parameter name
			if seg.value == "" {
				return fmt.Errorf("empty parameter name in pattern %q", p.raw)
			}
			// Check for duplicate parameter names
			if seenParams[seg.value] {
				return fmt.Errorf("duplicate parameter name %q in pattern %q", seg.value, p.raw)
			}
			seenParams[seg.value] = true
		} else {
			// Check for wildcard in static segment
			if seg.value == "*" {
				return fmt.Errorf("wildcard * not supported in pattern %q", p.raw)
			}
		}
	}

	return nil
}

// match attempts to match a request path against this pattern.
// Returns a map of captured parameters and a boolean indicating success.
// Returns (nil, false) if the path does not match the pattern.
func (p *pattern) match(path string) (map[string]string, bool) {
	// Split the request path by "/" and skip the leading empty string
	parts := strings.Split(path, "/")
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}

	// Handle root path: "/" becomes [""] after split and skip
	if len(parts) == 0 {
		parts = []string{""}
	}

	// Check if segment counts match
	if len(parts) != len(p.segments) {
		return nil, false
	}

	params := make(map[string]string)
	for i, seg := range p.segments {
		if seg.isParam {
			// Parameter segment: capture the value
			// Reject empty parameter values
			if parts[i] == "" {
				return nil, false
			}
			params[seg.value] = parts[i]
		} else {
			// Static segment: must match exactly
			if parts[i] != seg.value {
				return nil, false
			}
		}
	}

	return params, true
}

// specificity calculates the specificity score of this pattern.
// Exact segments score 2 points, parameter segments score 1 point.
// Higher specificity patterns should be matched first.
func (p *pattern) specificity() int {
	score := 0
	for _, seg := range p.segments {
		if seg.isParam {
			score += 1
		} else {
			score += 2
		}
	}
	return score
}
