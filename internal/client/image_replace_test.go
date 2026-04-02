package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestIsLocalImagePath(t *testing.T) {
	// Create a temp image file for testing
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.png")
	if err := os.WriteFile(tmpFile, []byte("fake png"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"empty string", "", false},
		{"non-image extension", "/tmp/file.txt", false},
		{"png but not exists", "/nonexistent/image.png", false},
		{"existing png", tmpFile, true},
		{"directory", tmpDir, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLocalImagePath(tt.path)
			if got != tt.want {
				t.Errorf("isLocalImagePath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestReplaceLocalImagesInPost_NoImages(t *testing.T) {
	input := `{"zh_cn":{"title":"test","content":[[{"tag":"text","text":"hello"}]]}}`
	result, err := ReplaceLocalImagesInPost(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != input {
		t.Errorf("expected no change, got %s", result)
	}
}

func TestReplaceLocalImagesInPost_InvalidJSON(t *testing.T) {
	input := "not json"
	result, err := ReplaceLocalImagesInPost(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != input {
		t.Errorf("expected input returned as-is, got %s", result)
	}
}

func TestReplaceLocalImagesInCard_Template(t *testing.T) {
	// Template references should not be modified
	input := `{"type":"template","data":{"template_id":"AAqk1xxx"}}`
	result, err := ReplaceLocalImagesInCard(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != input {
		t.Errorf("expected no change for template, got %s", result)
	}
}

func TestReplaceLocalImagesInCard_NoImages(t *testing.T) {
	input := `{"header":{"template":"blue","title":{"tag":"plain_text","content":"test"}},"elements":[{"tag":"markdown","content":"hello"}]}`
	result, err := ReplaceLocalImagesInCard(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != input {
		t.Errorf("expected no change, got %s", result)
	}
}

func TestReplaceLocalImagesInCard_WithRemoteImageKey(t *testing.T) {
	// img_key that looks like a remote key (img_v2_xxx) should not be touched
	card := map[string]interface{}{
		"header": map[string]interface{}{
			"template": "blue",
			"title":    map[string]interface{}{"tag": "plain_text", "content": "test"},
		},
		"elements": []interface{}{
			map[string]interface{}{
				"tag":     "img",
				"img_key": "img_v2_abc123",
				"alt":     map[string]interface{}{"tag": "plain_text", "content": "photo"},
			},
		},
	}
	inputBytes, _ := json.Marshal(card)
	input := string(inputBytes)

	result, err := ReplaceLocalImagesInCard(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != input {
		t.Errorf("expected no change for remote img_key, got %s", result)
	}
}
