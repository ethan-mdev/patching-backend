package handlers

import (
	"archive/zip"
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
	if err := json.NewEncoder(w).Encode(h.Manifest); err != nil {
		log.Printf("Error encoding manifest: %v", err)
		http.Error(w, "Failed to encode manifest", http.StatusInternalServerError)
		return
	}
}

func (h *PatchHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"version": h.Manifest.Version,
	})
}

func (h *PatchHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.PathValue("path")
	fullPath := filepath.Join(h.FilesRoot, filepath.Clean(filePath))

	// Security: prevent directory traversal
	if !filepath.HasPrefix(fullPath, h.FilesRoot) {
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

func (h *PatchHandler) DownloadBatch(w http.ResponseWriter, r *http.Request) {
	files := r.URL.Query()["files"]
	if len(files) == 0 {
		http.Error(w, "No files specified", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"patch.zip\"")

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	for _, file := range files {
		fullPath := filepath.Join(h.FilesRoot, filepath.Clean(file))

		// Security: prevent directory traversal
		if !filepath.HasPrefix(fullPath, h.FilesRoot) {
			log.Printf("Attempted directory traversal in batch: %s", file)
			continue
		}

		data, err := os.ReadFile(fullPath)
		if err != nil {
			log.Printf("Error reading file %s: %v", file, err)
			continue
		}

		writer, err := zipWriter.Create(file)
		if err != nil {
			log.Printf("Error creating zip entry for %s: %v", file, err)
			continue
		}

		if _, err := writer.Write(data); err != nil {
			log.Printf("Error writing file %s to zip: %v", file, err)
			continue
		}
	}
}

func (h *PatchHandler) CreatePatch(w http.ResponseWriter, r *http.Request) {
	version := r.PathValue("version")
	if version == "" {
		http.Error(w, "Version is required", http.StatusBadRequest)
		return
	}

	log.Printf("Creating patch for version: %s", version)

	// TODO: Implement patch creation logic
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
