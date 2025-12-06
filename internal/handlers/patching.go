package handlers

import (
	"archive/zip"
	"encoding/json"
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
	json.NewEncoder(w).Encode(h.Manifest)
}

func (h *PatchHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"version": h.Manifest.Version})
}

func (h *PatchHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.PathValue("path")
	fullPath := filepath.Join(h.FilesRoot, filepath.Clean(filePath))

	http.ServeFile(w, r, fullPath)
}

func (h *PatchHandler) DownloadBatch(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Files []string `json:"files"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=patch.zip")

	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, file := range request.Files {
		fullPath := filepath.Join(h.FilesRoot, filepath.Clean(file))

		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue // or handle error
		}

		fw, err := zw.Create(file)
		if err != nil {
			continue
		}
		fw.Write(data)
	}
}

func (h *PatchHandler) CreatePatch(w http.ResponseWriter, r *http.Request) {
	version := r.PathValue("version")
	if version == "" {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}

	newManifest, err := manifest.GenerateManifest(version, h.FilesRoot)
	if err != nil {
		http.Error(w, "failed to create patch: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := newManifest.Save(h.FilesRoot); err != nil {
		http.Error(w, "failed to save manifest: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.Manifest = newManifest

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "created",
		"version": version,
	})
}
