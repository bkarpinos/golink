package storage

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/bkarpinos/golink/internal/link"

	"github.com/fsnotify/fsnotify"
)

// JSONStorage implements link storage using a JSON file
type JSONStorage struct {
	filePath string
	links    map[string]*link.Link
	mutex    sync.RWMutex
}

// watchFile monitors the JSON file for changes and reloads when detected
func (s *JSONStorage) watchFile() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Error creating file watcher: %v", err)
		return
	}
	defer watcher.Close()

	// Watch the directory containing our file
	dir := filepath.Dir(s.filePath)
	if err := watcher.Add(dir); err != nil {
		log.Printf("Error watching directory %s: %v", dir, err)
		return
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// If our file was modified
			if event.Name == s.filePath && (event.Op&fsnotify.Write == fsnotify.Write) {
				// Give the file system a moment to complete the write
				time.Sleep(100 * time.Millisecond)

				s.mutex.Lock()
				err := s.load()
				s.mutex.Unlock()

				if err != nil {
					log.Printf("Error reloading links: %v", err)
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

// NewJSONStorage creates a new JSONStorage
func NewJSONStorage(filePath string) (*JSONStorage, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	storage := &JSONStorage{
		filePath: absPath,
		links:    make(map[string]*link.Link),
	}

	// Load existing data if file exists
	if _, err := os.Stat(absPath); !os.IsNotExist(err) {
		if err := storage.load(); err != nil {
			return nil, err
		}
	}

	// Start the file watcher in a goroutine
	go storage.watchFile()

	return storage, nil
}

// Save persists links to the JSON file (for external use)
func (s *JSONStorage) Save() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.saveWithoutLock()
}

// load reads links from the JSON file
func (s *JSONStorage) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	// s.mutex.Lock()
	// defer s.mutex.Unlock()

	// return json.Unmarshal(data, &s.links)

	// Create a temporary map to load the data
	tempLinks := make(map[string]*link.Link)

	// If the file is empty, just use an empty map
	if len(data) == 0 {
		s.links = tempLinks
		return nil
	}

	// Unmarshal JSON into the temporary map
	if err := json.Unmarshal(data, &tempLinks); err != nil {
		return err
	}

	// Replace the links map with our newly loaded data
	s.links = tempLinks
	return nil
}

// Create adds a new link
func (s *JSONStorage) Create(l *link.Link) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.links[l.Alias]; exists {
		return errors.New("link alias already exists")
	}

	s.links[l.Alias] = l
	// Don't call Save() while holding the lock
	return s.saveWithoutLock() // Call a private method that doesn't try to acquire the lock again
}

// saveWithoutLock saves without acquiring the lock (to be used internally)
func (s *JSONStorage) saveWithoutLock() error {
	data, err := json.MarshalIndent(s.links, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(s.filePath, data, 0644)
	return err
}

// Get retrieves a link by alias
func (s *JSONStorage) Get(alias string) (*link.Link, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	l, exists := s.links[alias]
	if !exists {
		return nil, errors.New("link not found")
	}

	return l, nil
}

// List returns all links
func (s *JSONStorage) List() []*link.Link {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make([]*link.Link, 0, len(s.links))
	for _, l := range s.links {
		result = append(result, l)
	}

	return result
}

// Update modifies an existing link
func (s *JSONStorage) Update(l *link.Link) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.links[l.Alias]; !exists {
		return errors.New("link not found")
	}

	s.links[l.Alias] = l
	return s.saveWithoutLock() // Use the internal method
}

// Delete removes a link
func (s *JSONStorage) Delete(alias string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.links[alias]; !exists {
		return errors.New("link not found")
	}

	delete(s.links, alias)
	return s.saveWithoutLock()
}
