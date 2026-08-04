package status

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var ErrNotFound = errors.New("runtime status not found")

type RuntimeEvent struct {
	At      time.Time `json:"at"`
	Message string    `json:"message"`
}

type ActivitySample struct {
	ProgramName string  `json:"program_name"`
	WindowTitle string  `json:"window_title"`
	IdleSeconds float64 `json:"idle_seconds"`
}

type RuntimeStatus struct {
	StartedAt              *time.Time      `json:"started_at,omitempty"`
	StoppedAt              *time.Time      `json:"stopped_at,omitempty"`
	LastSampleAt           *time.Time      `json:"last_sample_at,omitempty"`
	LastSuccessfulSampleAt *time.Time      `json:"last_successful_sample_at,omitempty"`
	SampleCount            int64           `json:"sample_count"`
	SelectedBackend        string          `json:"selected_backend"`
	ActiveBackend          string          `json:"active_backend"`
	CurrentWorkLogPath     string          `json:"current_work_log_path"`
	LastSuccessfulSample   *ActivitySample `json:"last_successful_sample,omitempty"`
	LatestWarning          *RuntimeEvent   `json:"latest_warning,omitempty"`
	LatestError            *RuntimeEvent   `json:"latest_error,omitempty"`
}

type Store struct {
	path string
}

func NewStore(path string) Store {
	return Store{path: path}
}

func (s Store) Path() string {
	return s.path
}

func (s Store) Read() (RuntimeStatus, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RuntimeStatus{}, ErrNotFound
		}
		return RuntimeStatus{}, fmt.Errorf("read runtime status %s: %w", s.path, err)
	}

	var runtimeStatus RuntimeStatus
	if err := json.Unmarshal(data, &runtimeStatus); err != nil {
		return RuntimeStatus{}, fmt.Errorf("decode runtime status %s: %w", s.path, err)
	}
	return runtimeStatus, nil
}

func (s Store) Write(runtimeStatus RuntimeStatus) error {
	if s.path == "" {
		return fmt.Errorf("runtime status path is empty")
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create runtime status directory %s: %w", dir, err)
	}

	tempFile, err := os.CreateTemp(dir, ".status-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary runtime status file in %s: %w", dir, err)
	}
	tempPath := tempFile.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tempPath)
		}
	}()

	encoder := json.NewEncoder(tempFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(runtimeStatus); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("encode runtime status %s: %w", tempPath, err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close runtime status temp file %s: %w", tempPath, err)
	}

	if err := os.Rename(tempPath, s.path); err != nil {
		return fmt.Errorf("replace runtime status %s: %w", s.path, err)
	}
	removeTemp = false
	return nil
}
