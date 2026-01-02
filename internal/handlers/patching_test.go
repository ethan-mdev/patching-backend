package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethan-mdev/patching-backend/internal/manifest"
)

func setupTestHandler(t *testing.T) (*PatchHandler, string) {
	t.Helper()

	// Create temp directory with test files
	tmpDir, err := os.MkdirTemp("", "patching-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create test files
	testFile := filepath.Join(tmpDir, "game.exe")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	m := &manifest.Manifest{
		Version: "1.0.0",
		Files: []manifest.FileHash{
			{FileName: "game.exe", Directory: "./", Hash: "abc123"},
			{FileName: "data.pak", Directory: "./", Hash: "def456"},
		},
	}

	return NewPatchHandler(m, tmpDir), tmpDir
}

func TestGetManifest(t *testing.T) {
	h, tmpDir := setupTestHandler(t)
	defer os.RemoveAll(tmpDir)

	req := httptest.NewRequest(http.MethodGet, "/manifest", nil)
	w := httptest.NewRecorder()

	h.GetManifest(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected content-type application/json, got %s", ct)
	}

	var m manifest.Manifest
	if err := json.NewDecoder(w.Body).Decode(&m); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if m.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", m.Version)
	}

	if len(m.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(m.Files))
	}
}

func TestGetManifest_NotModified(t *testing.T) {
	h, tmpDir := setupTestHandler(t)
	defer os.RemoveAll(tmpDir)

	req := httptest.NewRequest(http.MethodGet, "/manifest", nil)
	req.Header.Set("If-None-Match", "1.0.0")
	w := httptest.NewRecorder()

	h.GetManifest(w, req)

	if w.Code != http.StatusNotModified {
		t.Errorf("expected status 304, got %d", w.Code)
	}
}

func TestDownloadFile(t *testing.T) {
	h, tmpDir := setupTestHandler(t)
	defer os.RemoveAll(tmpDir)

	req := httptest.NewRequest(http.MethodGet, "/files/game.exe", nil)
	req.SetPathValue("path", "game.exe")
	w := httptest.NewRecorder()

	h.DownloadFile(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if body := w.Body.String(); body != "test content" {
		t.Errorf("expected 'test content', got '%s'", body)
	}
}

func TestDownloadFile_NotFound(t *testing.T) {
	h, tmpDir := setupTestHandler(t)
	defer os.RemoveAll(tmpDir)

	req := httptest.NewRequest(http.MethodGet, "/files/missing.exe", nil)
	req.SetPathValue("path", "missing.exe")
	w := httptest.NewRecorder()

	h.DownloadFile(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestDownloadFile_DirectoryTraversal(t *testing.T) {
	h, tmpDir := setupTestHandler(t)
	defer os.RemoveAll(tmpDir)

	attacks := []string{
		"../etc/passwd",
		"..\\windows\\system32",
		"foo/../../etc/passwd",
	}

	for _, path := range attacks {
		req := httptest.NewRequest(http.MethodGet, "/files/"+path, nil)
		req.SetPathValue("path", path)
		w := httptest.NewRecorder()

		h.DownloadFile(w, req)

		if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
			t.Errorf("path %q: expected 400 or 404, got %d", path, w.Code)
		}
	}
}

func TestVerifyFiles_AllValid(t *testing.T) {
	h, tmpDir := setupTestHandler(t)
	defer os.RemoveAll(tmpDir)

	clientFiles := map[string]string{
		"game.exe": "abc123",
		"data.pak": "def456",
	}
	body, _ := json.Marshal(clientFiles)

	req := httptest.NewRequest(http.MethodPost, "/verify", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.VerifyFiles(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	if valid, ok := result["valid"].(bool); !ok || !valid {
		t.Errorf("expected valid=true, got %v", result["valid"])
	}
}

func TestVerifyFiles_Mismatch(t *testing.T) {
	h, tmpDir := setupTestHandler(t)
	defer os.RemoveAll(tmpDir)

	clientFiles := map[string]string{
		"game.exe": "wronghash",
		"data.pak": "def456",
	}
	body, _ := json.Marshal(clientFiles)

	req := httptest.NewRequest(http.MethodPost, "/verify", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.VerifyFiles(w, req)

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	if valid, ok := result["valid"].(bool); !ok || valid {
		t.Errorf("expected valid=false, got %v", result["valid"])
	}

	mismatches := result["mismatches"].([]interface{})
	if len(mismatches) != 1 || mismatches[0] != "game.exe" {
		t.Errorf("expected mismatches=[game.exe], got %v", mismatches)
	}
}

func TestVerifyFiles_Missing(t *testing.T) {
	h, tmpDir := setupTestHandler(t)
	defer os.RemoveAll(tmpDir)

	clientFiles := map[string]string{
		"game.exe": "abc123",
		// missing data.pak
	}
	body, _ := json.Marshal(clientFiles)

	req := httptest.NewRequest(http.MethodPost, "/verify", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.VerifyFiles(w, req)

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	if valid, ok := result["valid"].(bool); !ok || valid {
		t.Errorf("expected valid=false, got %v", result["valid"])
	}

	missing := result["missing"].([]interface{})
	if len(missing) != 1 || missing[0] != "data.pak" {
		t.Errorf("expected missing=[data.pak], got %v", missing)
	}
}

func TestCreatePatch(t *testing.T) {
	h, tmpDir := setupTestHandler(t)
	defer os.RemoveAll(tmpDir)

	req := httptest.NewRequest(http.MethodPost, "/patches/1.0.1", nil)
	req.SetPathValue("version", "1.0.1")
	w := httptest.NewRecorder()

	h.CreatePatch(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var result map[string]string
	json.NewDecoder(w.Body).Decode(&result)

	if result["version"] != "1.0.1" {
		t.Errorf("expected version 1.0.1, got %s", result["version"])
	}
}
