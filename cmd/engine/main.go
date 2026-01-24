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

type PatchRequest struct {
	Order domain.Order             `json:"order"`
	Patch []map[string]interface{} `json:"patch"`
}

func main() {
	e := echo.New()

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodPost, http.MethodPatch, http.MethodOptions, http.MethodGet},
		AllowHeaders: []string{echo.HeaderContentType, echo.HeaderAccept},
	}))

	loader := infrastructure.NewFileRuleLoader()
	executor := infrastructure.NewJsonLogicExecutor()

	executor.RegisterCustomOperator("allocate", infrastructure.CustomAllocate)
	executor.RegisterCustomOperator("round", infrastructure.CustomRound)

	engineSvc := usecase.NewEngineService(loader, executor)

	// Endpoints solicitados pelo CEO e Tech Lead
	e.POST("/orders", handleCalculate(engineSvc))
	e.PATCH("/orders", handlePatch(engineSvc))
	e.POST("/sales", handleSale(engineSvc))
	e.GET("/products", handleListProducts)

	e.Logger.Fatal(e.Start(":8080"))
}

// OPA PDP: Retorna Constraints conforme o authz.rego
func consultOPA(permission string) (bool, float64) {
	if permission == "order.discount.apply" {
		return true, 0.15 // Constraint: Máximo 15%
	}
	return false, 0
}

func handleCalculate(svc interfaces.EngineFacade) echo.HandlerFunc {
	return func(c echo.Context) error {
		var order domain.Order
		if err := c.Bind(&order); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Payload inválido"})
		}

		// Enforcement de Constraint OPA
		_, maxDiscount := consultOPA("order.discount.apply")
		if order.DiscountPercentage > maxDiscount {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error":   "OPA Constraint Violation",
				"reasons": []string{fmt.Sprintf("Desconto de %.0f%% excede o limite permitido pelo OPA (15%%).", order.DiscountPercentage*100)},
			})
		}

		result, err := svc.RunEngine(c.Request().Context(), order, "v1.2")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, result)
	}
}

func handlePatch(svc interfaces.EngineFacade) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req PatchRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid patch request"})
		}

		patchBytes, _ := json.Marshal(req.Patch)
		updatedOrder, err := infrastructure.ApplyOrderPatch(req.Order, patchBytes)
		if err != nil {
			return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}

		// Re-validar contra OPA após o Patch
		_, maxDiscount := consultOPA("order.discount.apply")
		if updatedOrder.DiscountPercentage > maxDiscount {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error":   "OPA Constraint Violation (Post-Patch)",
				"reasons": []string{"A alteração solicitada viola os limites de desconto."},
			})
		}

		result, err := svc.RunEngine(c.Request().Context(), updatedOrder, "v1.2")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, result)
	}
}

func handleSale(svc interfaces.EngineFacade) echo.HandlerFunc {
	return func(c echo.Context) error {
		var order domain.Order
		if err := c.Bind(&order); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Payload inválido"})
		}

		// Enforcement Final (Guards)
		result, _ := svc.RunEngine(c.Request().Context(), order, "v1.2")
		if len(result.GuardsHit) > 0 {
			return c.JSON(http.StatusForbidden, map[string]interface{}{"error": "Blocked by Guards", "guards": result.GuardsHit})
		}

		order.ID = "SALE-" + time.Now().Format("20060102150405")
		order.CorrelationID = "CORR-" + order.ID

		// Persistência
		var sales []domain.Order
		file, _ := os.ReadFile("data/db/sales.json")
		json.Unmarshal(file, &sales)
		sales = append(sales, order)
		data, _ := json.MarshalIndent(sales, "", "  ")
		os.WriteFile("data/db/sales.json", data, 0644)

		return c.JSON(http.StatusCreated, order)
	}
}

func handleListProducts(c echo.Context) error {
	data, _ := os.ReadFile("data/db/products.json")
	var products []interface{}
	json.Unmarshal(data, &products)
	return c.JSON(http.StatusOK, products)
}
