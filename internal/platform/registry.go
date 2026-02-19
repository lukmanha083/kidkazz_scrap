package platform

import (
	"fmt"
	"sync"
)

var (
	registry = make(map[string]Scraper)
	mu       sync.RWMutex
)

func Register(name string, scraper Scraper) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = scraper
}

func Get(name string) (Scraper, error) {
	mu.RLock()
	defer mu.RUnlock()
	s, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("platform %q not registered", name)
	}
	return s, nil
}

func List() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
