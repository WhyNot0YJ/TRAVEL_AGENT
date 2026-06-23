package main

import "testing"

func TestBuildPlanner(t *testing.T) {
	tests := []struct {
		name        string
		plannerType string
		wantErr     bool
	}{
		{name: "mock", plannerType: "mock"},
		{name: "eino", plannerType: "eino"},
		{name: "unknown", plannerType: "bad", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planner, err := buildPlanner(tt.plannerType)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("buildPlanner returned error: %v", err)
			}
			if planner == nil {
				t.Fatal("buildPlanner returned nil planner")
			}
		})
	}
}
