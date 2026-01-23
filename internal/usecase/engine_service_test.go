package usecase

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Victor-armando18/service-commercial/internal/domain"
	"github.com/Victor-armando18/service-commercial/internal/infrastructure"
)

func TestEngine_DeterministicExecution(t *testing.T) {
	// Ajuste de diretório para encontrar pkg/rules durante o teste
	wd, _ := os.Getwd()
	for !strings.HasSuffix(wd, "service-commercial") {
		wd = filepath.Dir(wd)
	}
	os.Chdir(wd)

	loader := infrastructure.NewFileRuleLoader()
	executor := infrastructure.NewJsonLogicExecutor()
	executor.RegisterCustomOperator("allocate", infrastructure.CustomAllocate)
	executor.RegisterCustomOperator("round", infrastructure.CustomRound)

	engine := NewEngineService(loader, executor)

	order := domain.Order{
		ID:                 "TEST-DET-001",
		Currency:           "AOA",
		Items:              []domain.OrderItem{{SKU: "PROD1", Value: 1000, Qty: 2}},
		DiscountPercentage: 0.10,
	}

	res1, err := engine.RunEngine(context.Background(), order, "v1.2")
	if err != nil {
		t.Fatalf("Erro ao carregar regras: %v", err)
	}

	res2, _ := engine.RunEngine(context.Background(), order, "v1.2")

	if res1.StateFragment["totalValue"] != res2.StateFragment["totalValue"] {
		t.Errorf("Não determinístico")
	}
}
