package main

import (
	"encoding/json"
	"net/http"
	"os"

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

	e.Use(middleware.CORS())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Inicialização seguindo estritamente as assinaturas de internal
	loader := infrastructure.NewFileRuleLoader() // Sem argumentos conforme seu erro
	executor := infrastructure.NewJsonLogicExecutor()

	// Registro de operadores conforme sua infraestrutura internal
	executor.RegisterCustomOperator("allocate", infrastructure.CustomAllocate)
	executor.RegisterCustomOperator("round", infrastructure.CustomRound)

	// Engine service utilizando usecase/internal
	engineSvc := usecase.NewEngineService(loader, executor)

	e.POST("/orders", handleCalculate(engineSvc))
	e.PATCH("/orders", handlePatch(engineSvc))
	e.GET("/products", handleListProducts)
	e.POST("/sales", handleSale(engineSvc))

	e.Logger.Fatal(e.Start(":8080"))
}

func handleCalculate(svc interfaces.EngineFacade) echo.HandlerFunc {
	return func(c echo.Context) error {
		var order domain.Order
		if err := c.Bind(&order); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
		}

		version := order.RulesVersion
		if version == "" {
			version = "v1.2"
		}

		result, err := svc.RunEngine(c.Request().Context(), order, version)
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
		// Utiliza a função ApplyOrderPatch de infrastructure que trabalha com domain.Order
		updatedOrder, err := infrastructure.ApplyOrderPatch(req.Order, patchBytes)
		if err != nil {
			return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}

		version := updatedOrder.RulesVersion
		if version == "" {
			version = "v1.2"
		}

		result, err := svc.RunEngine(c.Request().Context(), updatedOrder, version)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, result)
	}
}

func handleListProducts(c echo.Context) error {
	data, err := os.ReadFile("db/products.json")
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Products file not found"})
	}
	var products []interface{}
	json.Unmarshal(data, &products)
	return c.JSON(http.StatusOK, products)
}

func handleSale(svc interfaces.EngineFacade) echo.HandlerFunc {
	return func(c echo.Context) error {
		var order domain.Order
		if err := c.Bind(&order); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid sale data"})
		}

		result, err := svc.RunEngine(c.Request().Context(), order, "v1.2")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		if len(result.GuardsHit) > 0 {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error":  "Sale blocked",
				"guards": result.GuardsHit,
			})
		}

		// Marshalling do fragmento para a struct final de domínio
		finalJSON, _ := json.Marshal(result.StateFragment)
		var finalizedOrder domain.Order
		json.Unmarshal(finalJSON, &finalizedOrder)

		finalizedOrder.CorrelationID = "CORR-" + finalizedOrder.ID

		// Persistência simples
		var sales []domain.Order
		file, _ := os.ReadFile("db/sales.json")
		json.Unmarshal(file, &sales)
		sales = append(sales, finalizedOrder)
		newData, _ := json.MarshalIndent(sales, "", "  ")
		os.WriteFile("db/sales.json", newData, 0644)

		return c.JSON(http.StatusCreated, finalizedOrder)
	}
}
