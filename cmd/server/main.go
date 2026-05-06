package main

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rmc8/go-todo-simple-starter/internal/renderer"
)

func main() {
	e := echo.New()
	r, err := renderer.NewTemplateRenderer("templates/*.html")
	if err != nil {
		log.Fatalf("failed to load templates: %v", err)
	}
	e.Renderer = r
	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "layout", nil)
	})
	e.Logger.Fatal(e.Start(":8888"))
}
