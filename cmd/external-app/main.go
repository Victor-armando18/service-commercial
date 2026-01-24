package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Victor-armando18/service-commercial/pkg/engine"
)

type LocalFileLoader struct {
	BasePath string
}

func (l *LocalFileLoader) Load(ctx context.Context, version string) (*engine.RulePack, error) {
	path := filepath.Join(l.BasePath, fmt.Sprintf("%s_rules.json", version))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ficheiro de regras não encontrado [%s]: %w", path, err)
	}

	var pack engine.RulePack
	if err := json.Unmarshal(data, &pack); err != nil {
		return nil, fmt.Errorf("erro ao parsear JSON de regras: %w", err)
	}

	return &pack, nil
}

func main() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("   RULE ENGINE CLI - DIAGNOSTIC TOOL")
	fmt.Println(strings.Repeat("=", 60))

	loader := &LocalFileLoader{BasePath: "data/rules"}
	executor := engine.NewJsonLogicExecutor()
	service := engine.NewEngineService(loader, executor)

	// Exemplo de pedido para teste da v1.2
	order := engine.Order{
		ID:                 "ORD-CLI-2024",
		Currency:           "AOA",
		BaseValue:          0, // Será calculado pelo foreach na fase baseline
		DiscountPercentage: 0.1,
		RulesVersion:       "v1.2",
		Items: []engine.OrderItem{
			{SKU: "PROD-A", Value: 100, Qty: 2},
			{SKU: "PROD-B", Value: 50, Qty: 1},
		},
	}

	result, err := service.RunEngine(context.Background(), order, order.RulesVersion)
	if err != nil {
		fmt.Printf("\n❌ ERRO CRÍTICO: %v\n", err)
		os.Exit(1)
	}

	displayExecutionSummary(result)
}

func displayExecutionSummary(res *engine.EngineResult) {
	// 1. LOG DE EXECUÇÃO (O Caminho Percorrido)
	fmt.Println("\n[1. LOG DE EXECUÇÃO]")
	for _, step := range res.ExecutionLog {
		fmt.Printf("   [%-12s] Rule: %-20s -> %s\n",
			strings.ToUpper(step.Phase), step.RuleID, step.Message)
	}

	// 2. GUARDS (Validações de Segurança)
	fmt.Println("\n[2. GUARDS / BLOQUEIOS]")
	if len(res.GuardsHit) == 0 {
		fmt.Println("   ✅ Nenhuma violação detectada.")
	} else {
		for _, guard := range res.GuardsHit {
			fmt.Printf("   ⚠️  BLOQUEIO: [%s] Motivo: %s\n", guard.RuleID, guard.Context)
		}
	}

	// 3. STATE FRAGMENT (O Objeto Resultante)
	fmt.Println("\n[3. STATE FRAGMENT (Objeto Final)]")
	fragmentJSON, _ := json.MarshalIndent(res.StateFragment, "   ", "  ")
	fmt.Println(string(fragmentJSON))

	// 4. RESUMO FINANCEIRO RÁPIDO
	fmt.Println("\n[4. RESUMO RÁPIDO]")
	fmt.Printf("   Status:      %s\n", map[bool]string{true: "BLOQUEADO", false: "APROVADO"}[len(res.GuardsHit) > 0])
	fmt.Printf("   Delta:       %v (Alterações feitas pelo servidor)\n", res.ServerDelta)
	fmt.Printf("   Versão Rule: %s\n", res.RulesVersion)

	fmt.Println(strings.Repeat("=", 60))
}
