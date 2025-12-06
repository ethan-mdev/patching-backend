package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifest_CreatesNewManifest(t *testing.T) {
	dir := t.TempDir()

	// Create a test file to be included in manifest
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	if m.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", m.Version)
	}

	if len(m.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(m.Files))
	}

	if m.Files[0].FileName != "test.txt" {
		t.Errorf("expected filename test.txt, got %s", m.Files[0].FileName)
	}

	// Verify manifest.json was created
	manifestPath := filepath.Join(dir, "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Error("manifest.json was not created")
	}
}

func TestLoadManifest_LoadsExistingManifest(t *testing.T) {
	dir := t.TempDir()

	// Create an existing manifest
	existing := &Manifest{
		Version: "2.5.0",
		Files: []FileHash{
			{FileName: "existing.txt", Directory: dir, Hash: "abc123"},
		},
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		t.Fatalf("failed to write existing manifest: %v", err)
	}

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	if m.Version != "2.5.0" {
		t.Errorf("expected version 2.5.0, got %s", m.Version)
	}

	if len(m.Files) != 1 || m.Files[0].FileName != "existing.txt" {
		t.Error("did not load existing manifest correctly")
	}
}

func TestLoadManifest_InvalidDirectory(t *testing.T) {
	_, err := LoadManifest("/nonexistent/path")
	if err == nil {
		t.Error("expected error for invalid directory")
	}
}

func TestLoadManifest_IgnoresSubdirectories(t *testing.T) {
	dir := t.TempDir()

	// Create a file
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Create a subdirectory
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	if len(m.Files) != 1 {
		t.Errorf("expected 1 file (ignoring subdir), got %d", len(m.Files))
	}
}

func TestLoadManifest_HashConsistency(t *testing.T) {
	dir := t.TempDir()

	testFile := filepath.Join(dir, "consistent.txt")
	if err := os.WriteFile(testFile, []byte("same content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	m1, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("first LoadManifest failed: %v", err)
	}

	// Remove the cached manifest to force regeneration
	os.Remove(filepath.Join(dir, "manifest.json"))

	m2, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("second LoadManifest failed: %v", err)
	}

	if m1.Files[0].Hash != m2.Files[0].Hash {
		t.Error("hash should be consistent for same file content")
	}
}
