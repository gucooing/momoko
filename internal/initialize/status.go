package initialize

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	DataDir    = "data"
	MarkerFile = "initialized.json"
)

var (
	m  = &Marker{}
	mu sync.RWMutex
)

func init() {
	Re()
}

func Re() {
	content, err := os.ReadFile(MarkerPath())
	if err != nil {
		return
	}

	if err := json.Unmarshal(content, &m); err != nil {
		return
	}
}

type Marker struct {
	Initialized bool      `json:"initialized"`
	CreatedAt   time.Time `json:"created_at"`
}

type Database struct {
	Type   string `json:"type"`
	Driver string `json:"driver"`
	Source string `json:"source"`
}

func MarkerPath() string {
	return filepath.Join(DataDir, MarkerFile)
}

func IsInitialized() bool {
	mu.RLock()
	defer mu.RUnlock()

	return m.Initialized
}

func WriteMarker() error {
	if err := os.MkdirAll(DataDir, 0o755); err != nil {
		return err
	}

	m.Initialized = true
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now()
	}

	content, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')

	if err := os.WriteFile(MarkerPath(), content, 0o600); err != nil {
		return err
	}

	return nil
}
