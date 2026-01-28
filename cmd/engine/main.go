package main

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Victor-armando18/service-commercial/internal/domain"
	"github.com/Victor-armando18/service-commercial/internal/infrastructure"
	"github.com/Victor-armando18/service-commercial/internal/interfaces"
	"github.com/Victor-armando18/service-commercial/internal/usecase"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	idempotencyStorage = make(map[string][]byte)
	idempotencyMu      sync.RWMutex
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
		AllowHeaders: []string{echo.HeaderContentType, echo.HeaderAccept, "X-Tenant-ID", "Idempotency-Key", "X-Correlation-ID"},
	}))

	loader := infrastructure.NewFileRuleLoader()
	executor := infrastructure.NewJsonLogicExecutor()
	executor.RegisterCustomOperator("allocate", infrastructure.CustomAllocate)
	executor.RegisterCustomOperator("round", infrastructure.CustomRound)

	engineSvc := usecase.NewEngineService(loader, executor)

	e.POST("/orders", handleCalculate(engineSvc))
	e.POST("/orders/patch", handlePatch(engineSvc))
	e.POST("/sales", handleSale(engineSvc))

	e.Logger.Fatal(e.Start(":8080"))
}

func handlePatch(svc interfaces.EngineFacade) echo.HandlerFunc {
	return func(c echo.Context) error {
		tenantID := c.Request().Header.Get("X-Tenant-ID")
		idempotencyKey := c.Request().Header.Get("Idempotency-Key")

		if idempotencyKey != "" {
			idempotencyMu.RLock()
			if val, ok := idempotencyStorage[idempotencyKey]; ok {
				idempotencyMu.RUnlock()
				return c.JSONBlob(http.StatusOK, val)
			}
			idempotencyMu.RUnlock()
		}

		var req PatchRequest
		if err := c.Bind(&req); err != nil {
			return errorRFC7807(c, http.StatusBadRequest, "Payload Inválido", err.Error())
		}

		patchBytes, _ := json.Marshal(req.Patch)
		updatedOrder, err := infrastructure.ApplyOrderPatch(req.Order, patchBytes)
		if err != nil {
			return errorRFC7807(c, http.StatusUnprocessableEntity, "Erro no Patch", err.Error())
		}

		ctx := c.Request().Context()
		result, err := svc.RunEngine(ctx, updatedOrder, updatedOrder.RulesVersion)
		if err != nil {
			return errorRFC7807(c, http.StatusInternalServerError, "Erro de Execução", err.Error())
		}

		result.StateFragment["tenantId"] = tenantID

		if idempotencyKey != "" {
			respBytes, _ := json.Marshal(result)
			idempotencyMu.Lock()
			idempotencyStorage[idempotencyKey] = respBytes
			idempotencyMu.Unlock()
		}

		return c.JSON(http.StatusOK, result)
	}
}

func handleCalculate(svc interfaces.EngineFacade) echo.HandlerFunc {
	return func(c echo.Context) error {
		var order domain.Order
		if err := c.Bind(&order); err != nil {
			return errorRFC7807(c, http.StatusBadRequest, "Erro de Parsing", err.Error())
		}

		result, err := svc.RunEngine(c.Request().Context(), order, order.RulesVersion)
		if err != nil {
			return errorRFC7807(c, http.StatusInternalServerError, "Erro no Motor", err.Error())
		}
		return c.JSON(http.StatusOK, result)
	}
}

func handleSale(svc interfaces.EngineFacade) echo.HandlerFunc {
	return func(c echo.Context) error {
		var order domain.Order
		if err := c.Bind(&order); err != nil {
			return errorRFC7807(c, http.StatusBadRequest, "Venda Inválida", err.Error())
		}

		result, _ := svc.RunEngine(c.Request().Context(), order, order.RulesVersion)
		if len(result.GuardsHit) > 0 {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"type":   "https://dolphin.com/err/guard-violation",
				"title":  "Venda Bloqueada por Guardas",
				"status": 403,
				"detail": result.GuardsHit[0].Context,
			})
		}

		order.ID = "SALE-" + time.Now().Format("20060102150405")

		saveToJSON("data/db/sales.json", order)

		return c.JSON(http.StatusCreated, order)
	}
}

func errorRFC7807(c echo.Context, status int, title, detail string) error {
	return c.JSON(status, map[string]interface{}{
		"type":   "https://dolphin.com/errors",
		"title":  title,
		"status": status,
		"detail": detail,
	})
}

func saveToJSON(path string, data interface{}) {
	var list []interface{}
	file, _ := os.ReadFile(path)
	json.Unmarshal(file, &list)
	list = append(list, data)
	newContent, _ := json.MarshalIndent(list, "", "  ")
	os.WriteFile(path, newContent, 0644)
}
