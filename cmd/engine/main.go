package main

import (
	"context"
	"fmt"
	"log"

	"service-commercial/internal/domain/model"
	"service-commercial/internal/infrastructure/yaml"
	"service-commercial/internal/interfaces"
)

func main() {
	pack, err := yaml.LoadRulePack("rules/commercial-v1.yml")
	if err != nil {
		log.Fatal(err)
	}

	order := model.Order{
		ID: "ORDER-001",
		Items: []model.Item{
			{SKU: "SKU-1", Quantity: 2, Price: 100},
			{SKU: "SKU-2", Quantity: 1, Price: 300},
		},
	}

	result, err := interfaces.RunEngine(context.Background(), order, pack)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("STATE: %+v\n", result.StateFragment)
	fmt.Printf("DELTA: %+v\n", result.Delta)
	fmt.Printf("REASONS: %+v\n", result.Reasons)
	fmt.Printf("VERSION: %s\n", result.RulesVersion)
}
