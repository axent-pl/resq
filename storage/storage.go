package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"
)

type Storage[IT comparable, DT any] struct {
	path   string
	data   map[IT]DT
	dataMU sync.RWMutex
}

func NewStorage[IT comparable, DT any](path string) (*Storage[IT, DT], error) {
	s := &Storage[IT, DT]{
		path: path,
		data: make(map[IT]DT),
	}
	// Load data, ignore empty or non-existing file.
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	// If the file is empty, we're done.
	if fi, err := f.Stat(); err == nil && fi.Size() == 0 {
		return s, nil
	}

	// Decode JSON into the map. Treat EOF (e.g., whitespace-only file) as empty.
	dec := json.NewDecoder(f)
	if err := dec.Decode(&s.data); err != nil && err != io.EOF {
		return nil, fmt.Errorf("decode json from %s: %w", path, err)
	}

	return s, nil
}

func (s *Storage[IT, DT]) persist() error {
	dir := filepath.Dir(s.path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	// Write to a temp file first for atomic replace.
	tmpFile, err := os.CreateTemp(dir, filepath.Base(s.path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	// Ensure the temp file gets cleaned up on failure.
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	enc := json.NewEncoder(tmpFile)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s.data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("encode json: %w", err)
	}

	// Flush to disk before rename.
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Atomic replace.
	if err := os.Rename(tmpFile.Name(), s.path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

func (s *Storage[IT, DT]) List() ([]DT, error) {
	s.dataMU.RLock()
	defer s.dataMU.RUnlock()
	return slices.Collect(maps.Values(s.data)), nil
}

func (s *Storage[IT, DT]) Create(idx IT, d DT) error {
	s.dataMU.Lock()
	defer s.dataMU.Unlock()
	s.data[idx] = d
	return s.persist()
}

func (s *Storage[IT, DT]) Update(idx IT, d DT) error {
	s.dataMU.Lock()
	defer s.dataMU.Unlock()
	if _, ok := s.data[idx]; !ok {
		return fmt.Errorf("data not found at index %v", idx)
	}
	s.data[idx] = d
	return s.persist()
}
