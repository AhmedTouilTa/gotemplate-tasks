package generator

import (
	_ "embed"
	"gotemplates/todo/db/todoapp"
)

var (
	//go:embed templates/tasks.html
	tmplContent string
	//go:embed templates/task.html
	tmplTask string
	//go:embed templates/new_task.html
	tmplNewTask string
)

type TasksPage struct {
	Tasks []todoapp.Task
}

type TaskPage struct {
	Task todoapp.Task
}

// TemplateText is the root template. base.html/tasks.html/task.html only
// {{define}} the named templates "base", "content" and "task" — none of them
// emit anything on their own. The trailing {{template "base" .}} is the actual
// root body that kicks off rendering: base -> content -> task.
func (*TasksPage) TemplateText() string {
	return tmplContent + tmplTask + `{{template "content" .}}`
}

func (*TaskPage) TemplateText() string {
	return tmplNewTask + `{{template "content" .}}`
}
