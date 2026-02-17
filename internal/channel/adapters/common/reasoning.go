package common

import (
	"regexp"
	"strings"
)

// Patterns for complete <tag>...</tag> blocks (including multiline content).
// Go's regexp uses RE2 which does not support backreferences, so we use
// separate patterns for each tag name.
var (
	thinkCompletePattern      = regexp.MustCompile(`(?is)<think>.*?</think>`)
	reasoningCompletePattern  = regexp.MustCompile(`(?is)<reasoning>.*?</reasoning>`)
	reflectionCompletePattern = regexp.MustCompile(`(?is)<reflection>.*?</reflection>`)
)

// Patterns for unclosed opening tags at the end of a streaming buffer.
var (
	thinkPartialPattern      = regexp.MustCompile(`(?is)<think>[^<]*$`)
	reasoningPartialPattern  = regexp.MustCompile(`(?is)<reasoning>[^<]*$`)
	reflectionPartialPattern = regexp.MustCompile(`(?is)<reflection>[^<]*$`)
)

// stripCompleteBlocks removes all closed reasoning tag blocks from text.
func stripCompleteBlocks(text string) string {
	text = thinkCompletePattern.ReplaceAllString(text, "")
	text = reasoningCompletePattern.ReplaceAllString(text, "")
	text = reflectionCompletePattern.ReplaceAllString(text, "")
	return text
}

// stripPartialBlocks removes unclosed reasoning tags at the end of text.
func stripPartialBlocks(text string) string {
	text = thinkPartialPattern.ReplaceAllString(text, "")
	text = reasoningPartialPattern.ReplaceAllString(text, "")
	text = reflectionPartialPattern.ReplaceAllString(text, "")
	return text
}

// StripReasoningTags removes complete <think>...</think>, <reasoning>...</reasoning>,
// and <reflection>...</reflection> blocks from text. Use for final/complete text.
func StripReasoningTags(text string) string {
	result := stripCompleteBlocks(text)
	return strings.TrimSpace(result)
}

// StripReasoningTagsStreaming removes both complete and partial (unclosed) reasoning
// tag blocks from text. Use for streaming buffers where a tag may not yet be closed.
func StripReasoningTagsStreaming(text string) string {
	result := stripCompleteBlocks(text)
	result = stripPartialBlocks(result)
	return strings.TrimSpace(result)
}
