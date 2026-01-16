package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/Victor-armando18/service-commercial/internal/domain"
	"github.com/Victor-armando18/service-commercial/internal/infrastructure"
	"github.com/Victor-armando18/service-commercial/internal/usecase"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type PatchRequest struct {
	Order domain.Order            `json:"order"` // Estado atual conhecido pelo front
	Patch []domain.PatchOperation `json:"patch"` // Deltas (JSON Patch)
}

func main() {
	e := echo.New()

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodPost, http.MethodGet, http.MethodPatch},
	}))
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	loader := infrastructure.NewFileRuleLoader()
	executor := infrastructure.NewJsonLogicExecutor()

	// Registro de operadores customizados da seção 7.3 da doc
	executor.RegisterCustomOperator("allocate", infrastructure.CustomAllocate)
	executor.RegisterCustomOperator("round", infrastructure.CustomRound)

	engine := usecase.NewEngineService(loader, executor)

	// Endpoint autoritativo /validate (MVP conforme seção 14)
	e.POST("/orders", func(c echo.Context) error {
		var order domain.Order
		if err := c.Bind(&order); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		}

		version := c.QueryParam("version")
		if version == "" {
			version = "v1.0"
		}
		if !strings.HasPrefix(version, "v") {
			version = "v" + version
		}

		result, err := engine.RunEngine(c.Request().Context(), order, version)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, result)
	})

	e.PATCH("/orders", func(c echo.Context) error {
		var req PatchRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid Patch Request"})
		}

		// 1. Transformar as operações de patch em []byte para o helper
		patchBytes, _ := json.Marshal(req.Patch)

		// 2. Aplicar o Delta ao pedido original (Autoritativo)
		updatedOrder, err := infrastructure.ApplyOrderPatch(req.Order, patchBytes)
		if err != nil {
			return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}

		// 3. Rodar o motor com o pedido atualizado
		version := c.QueryParam("version")
		if version == "" {
			version = "v1.1"
		}

		result, err := engine.RunEngine(c.Request().Context(), updatedOrder, version)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// Adicionar metadados de controle da Doc
		result.RulesVersion = version

		// Retornar o StateFragment (Seção 4.3 da Doc)
		return c.JSON(http.StatusOK, result)
	})

	// No main.go
	e.GET("/products", func(c echo.Context) error {
		data, _ := os.ReadFile("db/products.json")
		var products []map[string]interface{}
		json.Unmarshal(data, &products)
		return c.JSON(http.StatusOK, products)
	})

	e.POST("/sales", func(c echo.Context) error {
		var order domain.Order
		if err := c.Bind(&order); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		}

		result, err := engine.RunEngine(c.Request().Context(), order, "v1.1")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Engine Error"})
		}

		// 1. Verificação de Guardas
		if len(result.GuardsHit) > 0 {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error":  "Venda Bloqueada",
				"reason": result.GuardsHit[0].Context,
			})
		}

		// 2. Cálculo de Fallback para Subtotal (Caso o motor retorne nil)
		var manualBase float64
		for _, item := range order.Items {
			manualBase += item.Value * float64(item.Qty)
		}

		// 3. Hidratação Segura (Evitando Panic e 0s indesejados)
		if val, ok := result.StateFragment["baseValue"].(float64); ok && val > 0 {
			order.BaseValue = val
		} else {
			order.BaseValue = manualBase // Fallback
		}

		if val, ok := result.StateFragment["totalValue"].(float64); ok {
			order.TotalValue = val
		} else {
			order.TotalValue = order.BaseValue // Se não houver taxas/descontos
		}

		// Mapear Impostos
		order.AppliedTaxes = make(map[string]float64)
		if vat, ok := result.StateFragment["appliedTaxes.VAT"].(float64); ok {
			order.AppliedTaxes["VAT"] = vat
		}

		order.RulesVersion = result.RulesVersion
		order.CorrelationID = "CORR-" + order.ID

		// Persistência
		var sales []domain.Order
		file, _ := os.ReadFile("db/sales.json")
		json.Unmarshal(file, &sales)
		sales = append(sales, order)

		newData, _ := json.MarshalIndent(sales, "", "  ")
		os.WriteFile("db/sales.json", newData, 0644)

		return c.JSON(http.StatusCreated, order)
	})

	e.Logger.Fatal(e.Start(":8080"))
}
