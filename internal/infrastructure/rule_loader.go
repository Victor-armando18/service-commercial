package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Victor-armando18/service-commercial/internal/domain"
	"github.com/Victor-armando18/service-commercial/internal/interfaces"
)

type FileRuleLoader struct{}

func NewFileRuleLoader() interfaces.RulePackLoader {
	return &FileRuleLoader{}
}

func (l *FileRuleLoader) Load(ctx context.Context, version string) (*domain.RulePackDefinition, error) {
	filename := fmt.Sprintf("%s_rules.json", version)
	// Assumindo que estamos executando a partir da raiz do projeto
	path := filepath.Join("pkg", "rules", filename)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read rule file %s: %w", path, err)
	}

	var def domain.RulePackDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rule definition: %w", err)
	}

	return &def, nil
}
