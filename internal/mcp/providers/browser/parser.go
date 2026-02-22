package browser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Element represents an interactive element from browser snapshot.
type Element struct {
	Ref         string `json:"ref"`          // @e1, @e2, etc.
	Tag         string `json:"tag"`          // button, input, a, etc.
	Text        string `json:"text"`         // visible text content
	Selector    string `json:"selector"`     // CSS selector
	Placeholder string `json:"placeholder"`  // for input elements
	Type        string `json:"type"`         // input type attribute
	Href        string `json:"href"`         // for links
}

// SnapshotResponse represents the JSON response from agent-browser snapshot.
type SnapshotResponse struct {
	Success  bool      `json:"success"`
	Elements []Element `json:"elements"`
	Count    int       `json:"count"`
	Message  string    `json:"message"`
}

// elementRefRegex matches @e1, @e2, etc. but not @e0 or @e01
var elementRefRegex = regexp.MustCompile(`^@e[1-9]\d*$`)

// parseSnapshotElements parses the JSON output from agent-browser snapshot
// and returns a slice of Element structs.
func parseSnapshotElements(stdout string) ([]Element, error) {
	if stdout == "" {
		return nil, fmt.Errorf("empty snapshot output")
	}

	var resp SnapshotResponse
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		// Try parsing as raw array for backward compatibility
		var elements []Element
		if err2 := json.Unmarshal([]byte(stdout), &elements); err2 != nil {
			return nil, fmt.Errorf("failed to parse snapshot JSON: %w (original error: %v)", err2, err)
		}
		return elements, nil
	}

	if !resp.Success {
		return nil, fmt.Errorf("snapshot failed: %s", resp.Message)
	}

	return resp.Elements, nil
}

// isValidElementRef checks if a selector is a valid @e reference.
// Valid refs: @e1, @e2, @e42, etc.
// Invalid: @, @e, @e0, @e01, e1, @e-1
func isValidElementRef(ref string) bool {
	if ref == "" {
		return false
	}
	return elementRefRegex.MatchString(ref)
}

// extractElementRefNumber extracts the numeric part from @eN reference.
// Returns the number and true if valid, otherwise 0 and false.
func extractElementRefNumber(ref string) (int, bool) {
	if !isValidElementRef(ref) {
		return 0, false
	}
	var num int
	if _, err := fmt.Sscanf(ref, "@e%d", &num); err != nil {
		return 0, false
	}
	if num <= 0 {
		return 0, false
	}
	return num, true
}

// findElementByRef finds an element in the snapshot by @e reference.
func findElementByRef(elements []Element, ref string) (*Element, error) {
	if !isValidElementRef(ref) {
		return nil, fmt.Errorf("invalid element reference: %s", ref)
	}

	for i := range elements {
		if elements[i].Ref == ref {
			return &elements[i], nil
		}
	}

	return nil, fmt.Errorf("element %s not found in snapshot", ref)
}

// isCSSSelector checks if a string looks like a CSS selector (not an @e ref).
func isCSSSelector(selector string) bool {
	if selector == "" {
		return false
	}
	if isValidElementRef(selector) {
		return false
	}
	// Common CSS selector patterns: tag, .class, #id, [attr], tag.class, etc.
	return strings.ContainsAny(selector, ".#[ ") || isHTMLTag(selector)
}

// isHTMLTag checks if a string is a common HTML tag name.
func isHTMLTag(s string) bool {
	commonTags := map[string]bool{
		"a": true, "button": true, "input": true, "div": true, "span": true,
		"form": true, "select": true, "textarea": true, "img": true, "p": true,
		"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"ul": true, "ol": true, "li": true, "table": true, "tr": true, "td": true,
	}
	return commonTags[strings.ToLower(s)]
}

// parseJSONResponse is a generic JSON parser for CLI responses.
// Returns the parsed map and any error.
func parseJSONResponse(stdout string) (map[string]any, error) {
	if stdout == "" {
		return nil, fmt.Errorf("empty response")
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}

	return response, nil
}

// parseActionbookSearchResults parses actionbook search results.
type ActionbookResult struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Tags        []string `json:"tags"`
	Score       float64 `json:"score"`
}

// ActionbookSearchResponse represents the search results.
type ActionbookSearchResponse struct {
	Success bool               `json:"success"`
	Query   string             `json:"query"`
	Results []ActionbookResult `json:"results"`
	Count   int                `json:"count"`
}

// parseActionbookSearch parses actionbook search JSON output.
func parseActionbookSearch(stdout string) (*ActionbookSearchResponse, error) {
	if stdout == "" {
		return nil, fmt.Errorf("empty actionbook search output")
	}

	var resp ActionbookSearchResponse
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse actionbook search: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("actionbook search failed")
	}

	return &resp, nil
}
