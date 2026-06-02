package generator

import (
	"gotemplates/todo/models"
	"html/template"
	"log"
	"strings"
)

var tmpl = template.Must(template.ParseGlob("generator/templates/*.html"))

func Tasks(tasks []models.Task) string {
	out := new(strings.Builder)

	if err := tmpl.ExecuteTemplate(out, "base", map[string]any{"Tasks": tasks}); err != nil {
		log.Printf("cannot render tasks template: %s", err.Error())
		return ""
	}

	return out.String()
}
