package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethan-mdev/patching-backend/internal/integrity"
)

type Manifest struct {
	Version string     `json:"version"`
	Files   []FileHash `json:"files"`
}

type FileHash struct {
	FileName  string `json:"fileName"`
	Directory string `json:"directory"`
	Hash      string `json:"hash"`
}

// LoadManifest loads a manifest from the given directory, or creates one if it doesn't exist.
func LoadManifest(dir string) (*Manifest, error) {
	manifestPath := filepath.Join(dir, "manifest.json")

	cached, err := loadFromFile(manifestPath)
	if err == nil {
		return cached, nil
	}

	m, err := GenerateManifest("1.0.0", dir)
	if err != nil {
		return nil, err
	}

	if err := saveToFile(manifestPath, m); err != nil {
		return nil, err
	}

	return m, nil
}

func GenerateManifest(version string, dir string) (*Manifest, error) {
	m := &Manifest{
		Version: version,
		Files:   []FileHash{},
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("unable to read directory: %v", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			filePath := filepath.Join(dir, file.Name())
			hash, err := integrity.ComputeFileHash(filePath)
			if err != nil {
				return nil, fmt.Errorf("unable to compute hash for file %s: %v", file.Name(), err)
			}
			m.Files = append(m.Files, FileHash{
				FileName:  file.Name(),
				Directory: dir,
				Hash:      hash,
			})
		}
	}

	return m, nil
}

func loadFromFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func saveToFile(path string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Save writes the manifest to the given directory as manifest.json
func (m *Manifest) Save(dir string) error {
	manifestPath := filepath.Join(dir, "manifest.json")
	return saveToFile(manifestPath, m)
}
