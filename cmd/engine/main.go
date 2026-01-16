package main

import (
	"net/http"
	"strings"

	"github.com/Victor-armando18/service-commercial/internal/domain"
	"github.com/Victor-armando18/service-commercial/internal/infrastructure"
	"github.com/Victor-armando18/service-commercial/internal/usecase"
	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	loader := infrastructure.NewFileRuleLoader()
	executor := infrastructure.NewJsonLogicExecutor()
	executor.RegisterCustomOperator("round", infrastructure.CustomRound)
	executor.RegisterCustomOperator("allocate", infrastructure.CustomAllocate)
	engine := usecase.NewEngineService(loader, executor)

	e.POST("/orders", func(c echo.Context) error {
		var order domain.Order
		if err := c.Bind(&order); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
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
