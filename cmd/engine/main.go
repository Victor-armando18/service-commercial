package main

import (
	"net/http"
	"strings"

	"github.com/Victor-armando18/service-commercial/internal/domain"
	"github.com/Victor-armando18/service-commercial/internal/infrastructure"
	"github.com/Victor-armando18/service-commercial/internal/usecase"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()
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

	e.Logger.Fatal(e.Start(":8080"))
}
