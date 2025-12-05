package manifest

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethan-mdev/patching-backend/internal/integrity"
)

type Manifest struct {
	Version string
	Files   []FileHash
}

type FileHash struct {
	FileName  string
	Directory string
	Hash      string
}

const manifestCachePath = "./manifest.json"

func NewManifest() *Manifest {
	return &Manifest{
		Version: "",
		Files:   []FileHash{},
	}
}

func LoadManifest() (*Manifest, error) {
	cached, err := loadFromFile(manifestCachePath)
	if err == nil {
		return cached, nil
	}

	m, err := GenerateManifest("1.0.0", "./files")
	if err != nil {
		return nil, err
	}

	if err := m.saveToFile(manifestCachePath); err != nil {
		return nil, err
	}

	return m, nil
}

func GenerateManifest(version string, filepath string) (*Manifest, error) {
	m := NewManifest()
	m.SetVersion(version)

	files, err := os.ReadDir(filepath)
	if err != nil {
		return nil, fmt.Errorf("unable to read directory: %v", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			hash, err := integrity.ComputeFileHash(fmt.Sprintf("%s/%s", filepath, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("unable to compute hash for file %s: %v", file.Name(), err)
			}
			m.AddFile(file.Name(), filepath, hash)
		}
	}

	return m, nil
}

func (m *Manifest) SetVersion(version string) {
	m.Version = version
}

func (m *Manifest) GetVersion() string {
	return m.Version
}

func (m *Manifest) AddFile(file string, directory string, hash string) {
	m.Files = append(m.Files, FileHash{
		FileName:  file,
		Directory: directory,
		Hash:      hash,
	})
}

func (m *Manifest) GetFiles() []FileHash {
	return m.Files
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

func (m *Manifest) saveToFile(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
