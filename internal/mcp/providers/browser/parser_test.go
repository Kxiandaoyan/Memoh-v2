package browser

import (
	"testing"
)

// ─── parseSnapshotElements Tests ────────────────────────────────────────────

func TestParseSnapshotElements_Success(t *testing.T) {
	json := `{
		"success": true,
		"elements": [
			{"ref":"@e1","tag":"button","text":"Submit","selector":"button.submit"},
			{"ref":"@e2","tag":"input","type":"text","selector":"input#username"}
		],
		"count": 2
	}`

	elements, err := parseSnapshotElements(json)
	if err != nil {
		t.Fatal(err)
	}

	if len(elements) != 2 {
		t.Errorf("expected 2 elements, got %d", len(elements))
	}

	if elements[0].Ref != "@e1" {
		t.Errorf("expected @e1, got %s", elements[0].Ref)
	}
	if elements[0].Tag != "button" {
		t.Errorf("expected button, got %s", elements[0].Tag)
	}
}

func TestParseSnapshotElements_ArrayFormat(t *testing.T) {
	json := `[
		{"ref":"@e1","tag":"button","text":"Submit"},
		{"ref":"@e2","tag":"input","type":"text"}
	]`

	elements, err := parseSnapshotElements(json)
	if err != nil {
		t.Fatal(err)
	}

	if len(elements) != 2 {
		t.Errorf("expected 2 elements, got %d", len(elements))
	}
}

func TestParseSnapshotElements_EmptyInput(t *testing.T) {
	_, err := parseSnapshotElements("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParseSnapshotElements_InvalidJSON(t *testing.T) {
	_, err := parseSnapshotElements("not json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseSnapshotElements_FailedResponse(t *testing.T) {
	json := `{"success": false, "message": "browser not initialized"}`

	_, err := parseSnapshotElements(json)
	if err == nil {
		t.Error("expected error for failed response")
	}
	if err != nil && err.Error() != "snapshot failed: browser not initialized" {
		t.Errorf("unexpected error: %v", err)
	}
}

// ─── isValidElementRef Tests ────────────────────────────────────────────────

func TestIsValidElementRef(t *testing.T) {
	tests := []struct {
		ref   string
		valid bool
	}{
		{"@e1", true},
		{"@e2", true},
		{"@e42", true},
		{"@e999", true},
		{"", false},
		{"@e", false},
		{"@e0", false},
		{"@e01", false},
		{"e1", false},
		{"@e-1", false},
		{"button", false},
		{"#id", false},
		{".class", false},
	}

	for _, tt := range tests {
		got := isValidElementRef(tt.ref)
		if got != tt.valid {
			t.Errorf("isValidElementRef(%q) = %v, want %v", tt.ref, got, tt.valid)
		}
	}
}

// ─── extractElementRefNumber Tests ──────────────────────────────────────────

func TestExtractElementRefNumber(t *testing.T) {
	tests := []struct {
		ref    string
		number int
		valid  bool
	}{
		{"@e1", 1, true},
		{"@e2", 2, true},
		{"@e42", 42, true},
		{"@e999", 999, true},
		{"@e0", 0, false},
		{"@e", 0, false},
		{"", 0, false},
		{"e1", 0, false},
		{"button", 0, false},
	}

	for _, tt := range tests {
		num, valid := extractElementRefNumber(tt.ref)
		if valid != tt.valid {
			t.Errorf("extractElementRefNumber(%q) valid = %v, want %v", tt.ref, valid, tt.valid)
		}
		if valid && num != tt.number {
			t.Errorf("extractElementRefNumber(%q) = %d, want %d", tt.ref, num, tt.number)
		}
	}
}

// ─── findElementByRef Tests ─────────────────────────────────────────────────

func TestFindElementByRef_Success(t *testing.T) {
	elements := []Element{
		{Ref: "@e1", Tag: "button", Text: "Submit"},
		{Ref: "@e2", Tag: "input", Type: "text"},
	}

	elem, err := findElementByRef(elements, "@e1")
	if err != nil {
		t.Fatal(err)
	}
	if elem.Tag != "button" {
		t.Errorf("expected button, got %s", elem.Tag)
	}
}

func TestFindElementByRef_NotFound(t *testing.T) {
	elements := []Element{
		{Ref: "@e1", Tag: "button"},
	}

	_, err := findElementByRef(elements, "@e99")
	if err == nil {
		t.Error("expected error for not found element")
	}
}

func TestFindElementByRef_InvalidRef(t *testing.T) {
	elements := []Element{
		{Ref: "@e1", Tag: "button"},
	}

	_, err := findElementByRef(elements, "button")
	if err == nil {
		t.Error("expected error for invalid ref")
	}
}

// ─── isCSSSelector Tests ────────────────────────────────────────────────────

func TestIsCSSSelector(t *testing.T) {
	tests := []struct {
		selector string
		isCSS    bool
	}{
		{"button.submit", true},
		{"#username", true},
		{".class-name", true},
		{"[type='text']", true},
		{"button", true},
		{"input", true},
		{"div span", true},
		{"@e1", false},
		{"@e2", false},
		{"", false},
	}

	for _, tt := range tests {
		got := isCSSSelector(tt.selector)
		if got != tt.isCSS {
			t.Errorf("isCSSSelector(%q) = %v, want %v", tt.selector, got, tt.isCSS)
		}
	}
}

// ─── isHTMLTag Tests ────────────────────────────────────────────────────────

func TestIsHTMLTag(t *testing.T) {
	tests := []struct {
		tag   string
		valid bool
	}{
		{"button", true},
		{"input", true},
		{"div", true},
		{"a", true},
		{"span", true},
		{"form", true},
		{"BUTTON", true}, // Case insensitive
		{"@e1", false},
		{"notag", false},
		{"", false},
	}

	for _, tt := range tests {
		got := isHTMLTag(tt.tag)
		if got != tt.valid {
			t.Errorf("isHTMLTag(%q) = %v, want %v", tt.tag, got, tt.valid)
		}
	}
}

// ─── parseJSONResponse Tests ────────────────────────────────────────────────

func TestParseJSONResponse_Success(t *testing.T) {
	json := `{"success": true, "data": "test"}`

	result, err := parseJSONResponse(json)
	if err != nil {
		t.Fatal(err)
	}

	if result["success"] != true {
		t.Error("expected success=true")
	}
	if result["data"] != "test" {
		t.Errorf("expected data=test, got %v", result["data"])
	}
}

func TestParseJSONResponse_EmptyInput(t *testing.T) {
	_, err := parseJSONResponse("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParseJSONResponse_InvalidJSON(t *testing.T) {
	_, err := parseJSONResponse("not json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// ─── parseActionbookSearch Tests ────────────────────────────────────────────

func TestParseActionbookSearch_Success(t *testing.T) {
	json := `{
		"success": true,
		"query": "login",
		"results": [
			{"id":"github-login","name":"GitHub Login","score":0.95},
			{"id":"gmail-login","name":"Gmail Login","score":0.85}
		],
		"count": 2
	}`

	result, err := parseActionbookSearch(json)
	if err != nil {
		t.Fatal(err)
	}

	if !result.Success {
		t.Error("expected success=true")
	}
	if result.Query != "login" {
		t.Errorf("expected query=login, got %s", result.Query)
	}
	if len(result.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Results))
	}
	if result.Results[0].ID != "github-login" {
		t.Errorf("expected github-login, got %s", result.Results[0].ID)
	}
}

func TestParseActionbookSearch_EmptyInput(t *testing.T) {
	_, err := parseActionbookSearch("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParseActionbookSearch_InvalidJSON(t *testing.T) {
	_, err := parseActionbookSearch("not json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseActionbookSearch_Failed(t *testing.T) {
	json := `{"success": false}`

	_, err := parseActionbookSearch(json)
	if err == nil {
		t.Error("expected error for failed search")
	}
}
