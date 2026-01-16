package main

import (
	"context"
	"fmt"

	"github.com/Victor-armando18/service-commercial/internal/domain"
	"github.com/Victor-armando18/service-commercial/internal/infrastructure"
	"github.com/Victor-armando18/service-commercial/internal/usecase"
)

func main() {
	loader := infrastructure.NewFileRuleLoader()
	executor := infrastructure.NewJsonLogicExecutor()

	executor.RegisterCustomOperator("round", infrastructure.CustomRound)
	executor.RegisterCustomOperator("allocate", infrastructure.CustomAllocate)

	engine := usecase.NewEngineService(loader, executor)
	ctx := context.Background()

	// Cenário usando OrderItems
	order := domain.Order{
		ID: "TARGET_ID_123",
		Items: []domain.OrderItem{
			{SKU: "IPHONE-15", Value: 1000.0, Qty: 1},
			{SKU: "CASE-PRO", Value: 50.0, Qty: 2},
		},
		DiscountPercentage: 0.10,
	}

	res, _ := engine.RunEngine(ctx, order, "v1.0")

	fmt.Printf("Pedido: %s\n", res.FinalOrder.ID)
	fmt.Printf("Itens Totais calculados: %d\n", res.FinalOrder.TotalItems)
	fmt.Printf("Valor Base calculado: %.2f\n", res.FinalOrder.BaseValue)
	fmt.Printf("Impostos Aplicados: %v\n", res.FinalOrder.AppliedTaxes)

	for _, log := range res.ExecutionLog {
		fmt.Printf(" -> [%s] Rule: %s applied\n", log.Phase, log.RuleID)
	}
}
