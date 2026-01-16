package usecase

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Victor-armando18/service-commercial/internal/domain"
	"github.com/Victor-armando18/service-commercial/internal/infrastructure"
)

func TestEngineService_FullFlow(t *testing.T) {
	// IMPORTANTE: Muda o diretório de trabalho para a raiz do projeto para o Loader encontrar os ficheiros
	// ou garante que o caminho é ajustado.
	wd, _ := os.Getwd()
	// Se estamos em internal/usecase, subimos dois níveis para chegar à raiz
	os.Chdir(filepath.Join(wd, "..", ".."))
	defer os.Chdir(wd)

	loader := infrastructure.NewFileRuleLoader()
	executor := infrastructure.NewJsonLogicExecutor()

	// Registrar operadores necessários para o teste
	executor.RegisterCustomOperator("allocate", infrastructure.CustomAllocate)

	engine := NewEngineService(loader, executor)

	t.Run("Cálculo Completo: Itens + Desconto + Impostos", func(t *testing.T) {
		order := domain.Order{
			ID: "TEST-PROD-001",
			Items: []domain.OrderItem{
				{SKU: "GOPRO", Value: 500.0, Qty: 2}, // Total 1000
			},
			DiscountPercentage: 0.10, // 10% de desconto
		}

		res, err := engine.RunEngine(context.Background(), order, "v1.0")
		if err != nil {
			t.Fatalf("Erro ao executar engine: %v", err)
		}

		// 1000 - 10% = 900
		if res.FinalOrder.BaseValue != 900.0 {
			t.Errorf("BaseValue incorreto. Esperado 900, obtido %.2f", res.FinalOrder.BaseValue)
		}

		// 20% de 900 = 180
		if res.FinalOrder.AppliedTaxes["VAT"] != 180.0 {
			t.Errorf("VAT incorreto. Esperado 180, obtido %.2f", res.FinalOrder.AppliedTaxes["VAT"])
		}
	})
}
