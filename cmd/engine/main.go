package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Victor-armando18/service-commercial/internal/domain"
	"github.com/Victor-armando18/service-commercial/internal/infrastructure"
	"github.com/Victor-armando18/service-commercial/internal/interfaces"
	"github.com/Victor-armando18/service-commercial/internal/usecase"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()

	// Middleware de segurança e CORS (Fundamental para PATCH/POST do Front)
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodPost, http.MethodPatch, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderContentType, echo.HeaderAccept},
	}))

	loader := infrastructure.NewFileRuleLoader()
	executor := infrastructure.NewJsonLogicExecutor()

	// Registro de operadores especializados para o pipeline de 6 fases
	executor.RegisterCustomOperator("round", infrastructure.CustomRound)

	engineSvc := usecase.NewEngineService(loader, executor)

	// Endpoints Profissionais
	e.POST("/orders", handleCalculate(engineSvc)) // Reconciliação e Cálculo
	e.POST("/sales", handleSale(engineSvc))       // Enforcement Final e Persistência

	e.Logger.Fatal(e.Start(":8080"))
}

// PDP Mock: Consulta OPA para obter Capabilities e Constraints
// Baseado no arquivo authz.rego fornecido: "order.discount.apply"
func consultOPA(permission string) (bool, float64) {
	// Simula decisão do OPA: Usuário tem permissão, mas o limite (constraint) é 15% (0.15)
	if permission == "order.discount.apply" {
		return true, 0.15
	}
	return false, 0
}

func handleCalculate(svc interfaces.EngineFacade) echo.HandlerFunc {
	return func(c echo.Context) error {
		var order domain.Order
		if err := c.Bind(&order); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Payload inválido"})
		}

		// 1. OPA: Validar Constraint de Desconto (PEP Enforcement)
		allowed, maxDiscount := consultOPA("order.discount.apply")
		if !allowed || order.DiscountPercentage > maxDiscount {
			// Retornamos um erro estruturado que o front usará para bloquear a UI
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error":   "Constraint Violation",
				"reasons": []string{fmt.Sprintf("O desconto solicitado excede o limite permitido (Máx: %.0f%%)", maxDiscount*100)},
				"block":   true,
			})
		}

		// 2. Engine: Executar Pipeline por Fases (Baseline -> OrderAdjust -> Taxes -> Guards)
		result, err := svc.RunEngine(c.Request().Context(), order, order.RulesVersion)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// 3. Resposta rica com stateFragment e Guards
		return c.JSON(http.StatusOK, result)
	}
}

func handleSale(svc interfaces.EngineFacade) echo.HandlerFunc {
	return func(c echo.Context) error {
		var order domain.Order
		if err := c.Bind(&order); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Payload inválido"})
		}

		// Enforcement de Segurança Máxima: Re-executa Guards antes de persistir
		result, _ := svc.RunEngine(c.Request().Context(), order, order.RulesVersion)
		if len(result.GuardsHit) > 0 {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error":  "Security Guard Block",
				"guards": result.GuardsHit,
			})
		}

		// Idempotência e Metadados
		order.ID = "SALE-" + time.Now().Format("20060102150405")
		order.CorrelationID = "CORR-" + order.ID

		// Persistência em data/db/sales.json
		filePath := "data/db/sales.json"
		var sales []domain.Order
		fileData, _ := os.ReadFile(filePath)
		json.Unmarshal(fileData, &sales)
		sales = append(sales, order)

		finalJSON, _ := json.MarshalIndent(sales, "", "  ")
		os.WriteFile(filePath, finalJSON, 0644)

		return c.JSON(http.StatusCreated, order)
	}
}
