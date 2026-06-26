package main

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	_ "modernc.org/sqlite"

	"gotemplates/todo/db/todoapp"
	"gotemplates/todo/generator"
)

//go:embed db/schema.sql
var ddl string

//go:embed generator/templates/base.html
var homePage string

func main() {
	gctx := context.Background()

	db, err := sql.Open("sqlite", "test.db")

	if err != nil {
		log.Printf("could not start db %s", err.Error())
		return
	}

	if _, err := db.ExecContext(gctx, ddl); err != nil {
		log.Printf("could not exec start db %s", err.Error())
		return
	}

	queries := todoapp.New(db)

	// Create a Gin router with default middleware (logger and recovery)
	r := gin.Default()

	r.GET("/", func(ctx *gin.Context) {
		// retval, err := json.Marshal(tasks)

		// if err != nil {
		// 	log.Printf("marshall tasks %s", err.Error())
		// 	return
		// }
		FetchAndRenderHome(ctx, queries)
	})

	r.GET("/tasks", func(ctx *gin.Context) {
		// retval, err := json.Marshal(tasks)

		// if err != nil {
		// 	log.Printf("marshall tasks %s", err.Error())
		// 	return
		// }
		FetchAndRenderTasks(ctx, queries)
	})

	r.GET("/task", func(ctx *gin.Context) {
		id := ctx.Query("id")
		iid, _ := strconv.Atoi(id)
		_, err := queries.GetTask(ctx, int64(iid))

		if err != nil {
			log.Printf("could not get task %s", err.Error())
			ctx.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

	})

	r.POST("/add-task", func(ctx *gin.Context) {
		var form taskForm
		if err := ctx.ShouldBindWith(&form, binding.Form); err != nil {
			log.Printf("could not bind task %s", err.Error())
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		newTask, err := queries.CreateTask(ctx, todoapp.CreateTaskParams{
			Name:        form.Name,
			Description: sql.NullString{String: form.Description, Valid: form.Description != ""},
			Done:        form.Done,
		})

		if err != nil {
			log.Printf("could not create task %s", err.Error())
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not create task"})
			return
		}

		FetchAndRenderNewTask(ctx, newTask)
	})

	r.POST("/update-task", func(ctx *gin.Context) {
		idStr := ctx.PostForm("ID")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing or invalid id"})
			return
		}

		// sqlc's UpdateTask writes every column, so start from the current row
		// and override only the fields the form actually submitted. This keeps
		// a partial form (e.g. just toggling Done) from blanking the others.
		existing, err := queries.GetTask(ctx, int64(id))
		if err != nil {
			log.Printf("could not get task %s", err.Error())
			ctx.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		params := todoapp.UpdateTaskParams{
			ID:          existing.ID,
			Name:        existing.Name,
			Description: existing.Description,
			Done:        existing.Done,
		}
		if v, ok := ctx.GetPostForm("Name"); ok {
			params.Name = v
		}
		if v, ok := ctx.GetPostForm("Description"); ok {
			params.Description = sql.NullString{String: v, Valid: v != ""}
		}
		if v, ok := ctx.GetPostForm("Done"); ok {
			params.Done = v == "true"
		}

		if _, err = queries.UpdateTask(ctx, params); err != nil {
			log.Printf("couldnt update task %s", err.Error())
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not update task"})
			return
		}

	})

	r.POST("/delete-task", func(ctx *gin.Context) {
		id := ctx.Query("id")
		iid, _ := strconv.Atoi(id)
		err := queries.DeleteTask(ctx, int64(iid))

		if err != nil {
			log.Printf("could not delete task %s", err.Error())
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete task"})
			return
		}
	})

	// Start server on port 8080 (default)
	// Server will listen on 0.0.0.0:8080 (localhost:8080 on Windows)
	r.Run()
}

// taskForm holds the form fields with plain, form-bindable types. We bind this
// instead of the sqlc todoapp.Task row struct, whose sql.NullString column would
// make gin's form binder try (and fail) to json.Unmarshal the raw value.
type taskForm struct {
	ID          int64  `form:"ID"`
	Name        string `form:"Name"`
	Description string `form:"Description"`
	Done        bool   `form:"Done"`
}

func FetchAndRenderTasks(ctx *gin.Context, queries *todoapp.Queries) {
	tasks, err := queries.ListTasks(ctx)
	if err != nil {
		log.Printf("could not get tasks %s", err.Error())
		return
	}
	buf := bytes.Buffer{}
	err = generator.TasksTemplate.Render(&buf, &generator.TasksPage{
		Tasks: tasks,
	})
	if err != nil {
		log.Printf("could not render tasks %s", err.Error())
		return
	}
	ctx.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
}

func FetchAndRenderHome(ctx *gin.Context, queries *todoapp.Queries) {
	ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(homePage))
}

func FetchAndRenderNewTask(ctx *gin.Context, newTask todoapp.Task) {
	buf := bytes.Buffer{}

	err := generator.TaskTemplate.Render(&buf, &generator.TaskPage{
		Task: newTask,
	})

	if err != nil {
		log.Printf("could not render task %s", err.Error())
		return
	}

	println(buf.String())
	ctx.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
}
