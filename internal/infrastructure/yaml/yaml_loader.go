package yaml

import (
	"os"

	"service-commercial/internal/domain/engine"

	"gopkg.in/yaml.v3"
)

func LoadRulePack(path string) (engine.RulePack, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return engine.RulePack{}, err
	}

	var pack engine.RulePack
	if err := yaml.Unmarshal(data, &pack); err != nil {
		return engine.RulePack{}, err
	}
	return pack, nil
}
