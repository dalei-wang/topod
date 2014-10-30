package memkv

import (
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type MemStore struct {
	FuncMap map[string]interface{}
	sync.RWMutex
	store map[string]string
}

func NewMemStore() *MemStore {
	s := &MemStore{store: make(map[string]string)}
	s.FuncMap = map[string]interface{}{
		"exists": s.Exists,
		"ls":     s.List,
		"getv":   s.GetValue,
		"getvs":  s.GetAllValues,
		"gets":   s.GetAll,
	}
	return s
}

func (s *MemStore) Exists(key string) bool {
	_, ok := s.store[key]
	return ok
}

func (s *MemStore) List(filePath string) []string {
	m := make([]string, 0)
	for k, v := range s.store {
		if strings.HasPrefix(k, filePath) {
			m = append(m, v)
		}
	}
	return m
}

func (s *MemStore) GetValue(key string) string {
	v, ok := s.store[key]
	if ok {
		return v
	} else {
		return ""
	}
}

func (s *MemStore) GetAllValues(pattern string) []string {
	vs := make([]string, 0)
	s.RLock()
	defer s.RUnlock()
	for k, v := range s.store {
		m, err := filepath.Match(pattern, k)
		if err != nil {
			continue
		}
		if m {
			vs = append(vs, v)
		}
	}
	sort.Strings(vs)
	return vs
}

func (s *MemStore) GetAll(pattern string) map[string]string {
	m := make(map[string]string)
	s.RLock()
	defer s.RUnlock()
	for k, v := range s.store {
		match, err := filepath.Match(pattern, k)
		if err != nil {
			continue
		}
		if match {
			m[k] = v
		}
	}
	return m
}

func (s *MemStore) Set(key, value string) {
	s.Lock()
	defer s.Unlock()
	s.store[key] = value
}

func (s *MemStore) Clear() {
	s.Lock()
	defer s.Unlock()
	s.store = make(map[string]string)
}
