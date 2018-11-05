package main

import (
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"net/http"
)

func status(c echo.Context) error {
	jsonMap := map[string]string{
		"status": "Active",
	}
	return c.JSON(http.StatusOK, jsonMap)
}

func exec(c echo.Contect) error {
}

func main() {
	e := echo.New()

	// == middleware ==
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	// ================

	// == routing ==
	e.GET("/api/compiler/status", status)
	e.GET("/api/compiler/exec", exec)
	// =============

	e.Start(":4567")
}
