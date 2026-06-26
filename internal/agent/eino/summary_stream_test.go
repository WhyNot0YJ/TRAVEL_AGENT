package eino

import "testing"

func TestExtractSummarySoFar_NotYetReachedKey(t *testing.T) {
	if got := extractSummarySoFar(`{"title":"Hangzhou","days":[`); got != "" {
		t.Fatalf("expected empty before summary key reached, got %q", got)
	}
}

func TestExtractSummarySoFar_PartialString(t *testing.T) {
	raw := `{"title":"Hangzhou","summary":"从上海出`
	if got := extractSummarySoFar(raw); got != "从上海出" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractSummarySoFar_ClosesString(t *testing.T) {
	raw := `{"title":"H","summary":"完整总结","days":[]}`
	if got := extractSummarySoFar(raw); got != "完整总结" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractSummarySoFar_HandlesEscapes(t *testing.T) {
	raw := `{"summary":"a\nb\"c"`
	if got := extractSummarySoFar(raw); got != "a\nb\"c" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractSummarySoFar_TrimsIncompleteUTF8(t *testing.T) {
	// "出" in UTF-8 is E5 87 BA. Truncate to E5 87 (incomplete) and ensure the
	// extractor drops it instead of emitting a partial rune.
	full := []byte(`{"summary":"出`)
	truncated := full[:len(full)-1]
	if got := extractSummarySoFar(string(truncated)); got != "" {
		t.Fatalf("expected incomplete UTF-8 trimmed to empty, got %q (% x)", got, []byte(got))
	}
}

func TestExtractSummarySoFar_IgnoresNestedSummaryString(t *testing.T) {
	// A "summary" substring inside another string value (e.g. an item.reason)
	// must not be picked up — only the top-level summary key counts.
	raw := `{"title":"x","items":[{"reason":"my summary is short"}],"summary":"real"`
	if got := extractSummarySoFar(raw); got != "real" {
		t.Fatalf("got %q", got)
	}
}
