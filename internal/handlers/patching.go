package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ethan-mdev/patching-backend/internal/manifest"
)

type PatchHandler struct {
	Manifest  *manifest.Manifest
	FilesRoot string
}

func NewPatchHandler(m *manifest.Manifest, filesRoot string) *PatchHandler {
	return &PatchHandler{
		Manifest:  m,
		FilesRoot: filesRoot,
	}
}

func (h *PatchHandler) GetManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes
	w.Header().Set("ETag", h.Manifest.Version)

	// Check if client has current version
	if match := r.Header.Get("If-None-Match"); match == h.Manifest.Version {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	if err := json.NewEncoder(w).Encode(h.Manifest); err != nil {
		slog.Error("failed to encode manifest", "error", err)
		http.Error(w, "Failed to encode manifest", http.StatusInternalServerError)
		return
	}
}

func (h *PatchHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.PathValue("path")
	fullPath := filepath.Join(h.FilesRoot, filepath.Clean(filePath))

	// Security: prevent directory traversal
	relPath, err := filepath.Rel(h.FilesRoot, fullPath)
	if err != nil || relPath == ".." || len(relPath) > 2 && relPath[:3] == ".."+string(filepath.Separator) {
		slog.Warn("attempted directory traversal", "path", filePath)
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		slog.Debug("file not found", "path", fullPath)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, fullPath)
}

func (h *PatchHandler) CreatePatch(w http.ResponseWriter, r *http.Request) {
	version := r.PathValue("version")
	if version == "" {
		http.Error(w, "Version is required", http.StatusBadRequest)
		return
	}

	slog.Info("creating patch", "version", version)

	// 1. Scan files directory

	// 2. Calculate hashes
	// 3. Update manifest
	// 4. Save manifest

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Patch created successfully",
		"version": version,
	})
}

// VerifyFiles checks if client's file hashes match the server manifest
// Used for "Repair Game Files" functionality
func (h *PatchHandler) VerifyFiles(w http.ResponseWriter, r *http.Request) {
	var clientFiles map[string]string // fileName -> hash
	if err := json.NewDecoder(r.Body).Decode(&clientFiles); err != nil {
		slog.Error("failed to decode verify request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	mismatches := []string{}

	// Check each client file against manifest
	for fileName, clientHash := range clientFiles {
		found := false
		for _, serverFile := range h.Manifest.Files {
			if serverFile.FileName == fileName {
				found = true
				if serverFile.Hash != clientHash {
					mismatches = append(mismatches, fileName)
					slog.Debug("hash mismatch", "file", fileName, "client", clientHash, "server", serverFile.Hash)
				}
				break
			}
		}

		if !found {
			slog.Debug("client has unknown file", "file", fileName)
		}
	}

	// Check for missing files
	missing := []string{}
	for _, serverFile := range h.Manifest.Files {
		if _, exists := clientFiles[serverFile.FileName]; !exists {
			missing = append(missing, serverFile.FileName)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":      len(mismatches) == 0 && len(missing) == 0,
		"mismatches": mismatches,
		"missing":    missing,
	})
}
