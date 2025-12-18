package handlers

import (
	"encoding/json"
	"log"
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
		log.Printf("Error encoding manifest: %v", err)
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
		log.Printf("Attempted directory traversal: %s", filePath)
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		log.Printf("File not found: %s", fullPath)
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

	log.Printf("Creating patch for version: %s", version)

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
		log.Printf("Error decoding verify request: %v", err)
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
					log.Printf("Hash mismatch for %s: client=%s server=%s",
						fileName, clientHash, serverFile.Hash)
				}
				break
			}
		}

		if !found {
			log.Printf("Client has unknown file: %s", fileName)
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
