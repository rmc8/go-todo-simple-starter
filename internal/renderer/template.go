package renderer

import (
	"html/template"
	"io"

	"github.com/labstack/echo/v4"
)

type TemplateRenderer struct {
	templates *template.Template
}

func NewTemplateRenderer(pattern string) (*TemplateRenderer, error) {
	t, err := template.ParseGlob(pattern)
	if err != nil {
		return nil, err
	}
	return &TemplateRenderer{templates: t}, nil
}

func (r *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error  {
	return r.templates.ExecuteTemplate(w, name, data)
}