package generator

import (
	"github.com/tylermmorton/tmpl"
)

var TasksTemplate = tmpl.MustCompile(&TasksPage{})
var TaskTemplate = tmpl.MustCompile(&TaskPage{})
