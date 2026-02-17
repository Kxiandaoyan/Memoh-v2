package common

import "testing"

func TestStripReasoningTags(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "no tags",
			in:   "Hello world",
			want: "Hello world",
		},
		{
			name: "think tags",
			in:   "<think>Let me think about this...</think>Here is the answer.",
			want: "Here is the answer.",
		},
		{
			name: "reasoning tags",
			in:   "<reasoning>Step 1: analyze\nStep 2: conclude</reasoning>\nThe answer is 42.",
			want: "The answer is 42.",
		},
		{
			name: "reflection tags",
			in:   "Intro <reflection>hmm</reflection> rest of text",
			want: "Intro  rest of text",
		},
		{
			name: "multiline think block",
			in:   "<think>\nLine 1\nLine 2\nLine 3\n</think>\nActual response here.",
			want: "Actual response here.",
		},
		{
			name: "multiple think blocks",
			in:   "<think>first</think>Hello <think>second</think>World",
			want: "Hello World",
		},
		{
			name: "case insensitive",
			in:   "<THINK>uppercase</THINK>Result",
			want: "Result",
		},
		{
			name: "empty after stripping",
			in:   "<think>only thinking</think>",
			want: "",
		},
		{
			name: "whitespace around stripped",
			in:   "  <think>thinking</think>  result  ",
			want: "result",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripReasoningTags(tt.in)
			if got != tt.want {
				t.Errorf("StripReasoningTags(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestStripReasoningTagsStreaming(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "no tags",
			in:   "Hello world",
			want: "Hello world",
		},
		{
			name: "complete think block",
			in:   "<think>done thinking</think>Result here",
			want: "Result here",
		},
		{
			name: "partial/unclosed think tag",
			in:   "<think>still thinking... not closed yet",
			want: "",
		},
		{
			name: "partial think with preceding text",
			in:   "Some text<think>still going",
			want: "Some text",
		},
		{
			name: "complete block plus partial",
			in:   "<think>first</think>Middle text<think>not done",
			want: "Middle text",
		},
		{
			name: "partial reasoning tag",
			in:   "Hello <reasoning>analyzing",
			want: "Hello",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripReasoningTagsStreaming(tt.in)
			if got != tt.want {
				t.Errorf("StripReasoningTagsStreaming(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
