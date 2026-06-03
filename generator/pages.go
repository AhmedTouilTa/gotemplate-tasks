package generator

import (
	_ "embed"
	"gotemplates/todo/models"
)

var (
	//go:embed templates/base.html
	tmplBase string
	//go:embed templates/tasks.html
	tmplContent string
	//go:embed templates/task.html
	tmplTask string
)

type TasksPage struct {
	Tasks []models.Task
}

// TemplateText is the root template. base.html/tasks.html/task.html only
// {{define}} the named templates "base", "content" and "task" — none of them
// emit anything on their own. The trailing {{template "base" .}} is the actual
// root body that kicks off rendering: base -> content -> task.
func (*TasksPage) TemplateText() string {
	return tmplBase + tmplContent + tmplTask + `{{template "base" .}}`
}
