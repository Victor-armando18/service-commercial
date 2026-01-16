package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/Victor-armando18/service-commercial/internal/domain"
	"github.com/Victor-armando18/service-commercial/internal/interfaces"
)

type FileRuleLoader struct {
	cache map[string]*domain.RulePackDefinition
	mu    sync.RWMutex
}

func NewFileRuleLoader() interfaces.RulePackLoader {
	return &FileRuleLoader{
		cache: make(map[string]*domain.RulePackDefinition),
	}
}

func (l *FileRuleLoader) Load(ctx context.Context, version string) (*domain.RulePackDefinition, error) {
	l.mu.RLock()
	if def, ok := l.cache[version]; ok {
		l.mu.RUnlock()
		return def, nil
	}
	l.mu.RUnlock()

	l.mu.Lock()
	defer l.mu.Unlock()

	if def, ok := l.cache[version]; ok {
		return def, nil
	}

	filename := fmt.Sprintf("%s_rules.json", version)
	path := filepath.Join("pkg", "rules", filename)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler ficheiro: %w", err)
	}

	var def domain.RulePackDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("falha no unmarshal: %w", err)
	}

	l.cache[version] = &def
	return &def, nil
}
