package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManifest(t *testing.T) {
	m := NewManifest()

	if m.Version != "" {
		t.Errorf("expected empty version, got %s", m.Version)
	}

	if len(m.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(m.Files))
	}
}

func TestSetVersion(t *testing.T) {
	m := NewManifest()
	m.SetVersion("1.2.3")

	if m.GetVersion() != "1.2.3" {
		t.Errorf("expected version 1.2.3, got %s", m.GetVersion())
	}
}

func TestAddFile(t *testing.T) {
	m := NewManifest()
	m.AddFile("test.txt", "/data", "abc123")

	files := m.GetFiles()
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	if files[0].FileName != "test.txt" {
		t.Errorf("expected filename test.txt, got %s", files[0].FileName)
	}

	if files[0].Directory != "/data" {
		t.Errorf("expected directory /data, got %s", files[0].Directory)
	}

	if files[0].Hash != "abc123" {
		t.Errorf("expected hash abc123, got %s", files[0].Hash)
	}
}

func TestAddMultipleFiles(t *testing.T) {
	m := NewManifest()
	m.AddFile("file1.txt", "/dir1", "hash1")
	m.AddFile("file2.txt", "/dir2", "hash2")
	m.AddFile("file3.txt", "/dir3", "hash3")

	files := m.GetFiles()
	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d", len(files))
	}
}

func TestGenerateManifest(t *testing.T) {
	// Create temp directory with test files
	tempDir := t.TempDir()

	testFiles := map[string]string{
		"file1.txt": "content one",
		"file2.txt": "content two",
	}

	for name, content := range testFiles {
		path := filepath.Join(tempDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	m, err := GenerateManifest("1.0.0", tempDir)
	if err != nil {
		t.Fatalf("failed to generate manifest: %v", err)
	}

	if m.GetVersion() != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", m.GetVersion())
	}

	if len(m.GetFiles()) != 2 {
		t.Errorf("expected 2 files, got %d", len(m.GetFiles()))
	}

	// Verify hashes are not empty
	for _, f := range m.GetFiles() {
		if f.Hash == "" {
			t.Errorf("expected non-empty hash for %s", f.FileName)
		}
	}
}

func TestGenerateManifestInvalidPath(t *testing.T) {
	_, err := GenerateManifest("1.0.0", "/nonexistent/path")
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}

func TestSaveAndLoadManifest(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "manifest.json")

	// Create and save manifest
	m := NewManifest()
	m.SetVersion("2.0.0")
	m.AddFile("test.txt", "/data", "somehash")

	if err := m.saveToFile(cachePath); err != nil {
		t.Fatalf("failed to save manifest: %v", err)
	}

	// Load it back
	loaded, err := loadFromFile(cachePath)
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	if loaded.GetVersion() != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", loaded.GetVersion())
	}

	if len(loaded.GetFiles()) != 1 {
		t.Errorf("expected 1 file, got %d", len(loaded.GetFiles()))
	}

	if loaded.GetFiles()[0].FileName != "test.txt" {
		t.Errorf("expected filename test.txt, got %s", loaded.GetFiles()[0].FileName)
	}
}

func TestLoadFromFileInvalidPath(t *testing.T) {
	_, err := loadFromFile("/nonexistent/manifest.json")
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}

func TestLoadFromFileInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	badPath := filepath.Join(tempDir, "bad.json")

	if err := os.WriteFile(badPath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err := loadFromFile(badPath)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
