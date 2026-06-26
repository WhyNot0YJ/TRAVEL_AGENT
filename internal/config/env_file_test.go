package config

import "testing"

func TestParseEnvLine(t *testing.T) {
	key, value, ok := parseEnvLine(`TRAVEL_AGENT_LLM_API_KEY="secret"`)
	if !ok {
		t.Fatal("expected env line to parse")
	}
	if key != "TRAVEL_AGENT_LLM_API_KEY" || value != "secret" {
		t.Fatalf("unexpected parsed env: key=%q value=%q", key, value)
	}
}

func TestParseEnvLineSkipsComments(t *testing.T) {
	if _, _, ok := parseEnvLine("# comment"); ok {
		t.Fatal("comment should not parse")
	}
}
