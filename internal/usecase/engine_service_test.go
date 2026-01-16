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
	// 1. Ajuste do diretório de trabalho para localizar pkg/rules
	wd, _ := os.Getwd()
	// Sobe dois níveis (de internal/usecase para a raiz)
	root := filepath.Join(wd, "..", "..")
	if err := os.Chdir(root); err != nil {
		t.Fatalf("Não foi possível mudar para a raiz do projeto: %v", err)
	}
	defer os.Chdir(wd)

	// 2. Setup do Motor
	loader := infrastructure.NewFileRuleLoader()
	executor := infrastructure.NewJsonLogicExecutor()

	// Registrar operadores customizados exigidos pela Doc (Seção 7)
	executor.RegisterCustomOperator("allocate", infrastructure.CustomAllocate)
	executor.RegisterCustomOperator("round", infrastructure.CustomRound)

	engine := NewEngineService(loader, executor)

	t.Run("Deve validar o modelo de Cliente Inteligente com StateFragment", func(t *testing.T) {
		// Cenário:
		// Itens somam 1000.00
		// Desconto de 10% aplicado via 'orderAdjust' -> 900.00
		// Taxa de 20% aplicada via 'taxes' -> 180.00
		// Total Final Autoritativo -> 1080.00
		order := domain.Order{
			ID:            "TEST-DELTA-001",
			CorrelationID: "corr-123",
			RulesVersion:  "v1.0",
			Items: []domain.OrderItem{
				{SKU: "GOPRO", Value: 500.0, Qty: 2},
			},
			DiscountPercentage: 0.10,
		}

		res, err := engine.RunEngine(context.Background(), order, "v1.0")
		if err != nil {
			t.Fatalf("Erro ao executar engine: %v", err)
		}

		// 3. Validações baseadas na nova Doc (Seção 4.3 - State Fragment)

		// Verificar se o servidor marcou como ServerDelta (houve mudança autoritativa)
		if !res.ServerDelta {
			t.Error("Esperado ServerDelta=true, pois houve cálculos autoritativos")
		}

		// Verificar o Fragmento de Estado de BaseValue (1000 -> 900)
		if val, ok := res.StateFragment["baseValue"].(float64); !ok || val != 900.0 {
			t.Errorf("StateFragment['baseValue'] incorreto. Esperado 900.0, obtido %v", res.StateFragment["baseValue"])
		}

		// Verificar o Fragmento de Impostos (VAT)
		if val, ok := res.StateFragment["appliedTaxes.VAT"].(float64); !ok || val != 180.0 {
			t.Errorf("StateFragment['appliedTaxes.VAT'] incorreto. Esperado 180.0, obtido %v", res.StateFragment["appliedTaxes.VAT"])
		}

		// Verificar o Total Final recalculado (1080.0)
		if val, ok := res.StateFragment["totalValue"].(float64); !ok || val != 1080.0 {
			t.Errorf("StateFragment['totalValue'] incorreto. Esperado 1080.0, obtido %v", res.StateFragment["totalValue"])
		}

		// 4. Validar Logs de Auditoria (Seção 11 da Doc)
		foundAdjust := false
		for _, step := range res.ExecutionLog {
			if step.Phase == "orderAdjust" {
				foundAdjust = true
				break
			}
		}
		if !foundAdjust {
			t.Error("Fase 'orderAdjust' não foi executada ou logada")
		}
	})
}
