package main

import (
	"bytes"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "github.com/mattn/go-sqlite3"

	"gotemplates/todo/generator"
	"gotemplates/todo/models"
)

func main() {

	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})

	if err != nil {
		log.Printf("could not start db %s", err.Error())
		return
	}

	db.AutoMigrate(&models.Task{})

	// Create a Gin router with default middleware (logger and recovery)
	r := gin.Default()

	r.GET("/tasks", func(ctx *gin.Context) {
		tasks, err := gorm.G[models.Task](db).Find(ctx)

		if err != nil {
			log.Printf("could not get tasks %s", err.Error())
			return
		}

		// retval, err := json.Marshal(tasks)

		// if err != nil {
		// 	log.Printf("marshall tasks %s", err.Error())
		// 	return
		// }
		buf := bytes.Buffer{}
		err = generator.TasksTemplate.Render(&buf, &generator.TasksPage{
			Tasks: tasks,
		})
		if err != nil {
			log.Printf("could not render tasks %s", err.Error())
			return
		}
		log.Println(buf.String())
		ctx.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
	})

	r.GET("/task", func(ctx *gin.Context) {
		id := ctx.Query("id")

		_, err := gorm.G[models.Task](db).Where("id = ?", id).First(ctx)

		if err != nil {
			log.Printf("could not get task %s", err.Error())
			ctx.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		FetchAndRenderTasks(ctx, db)

	})

	r.POST("/add-task", func(ctx *gin.Context) {
		var task models.Task

		if err := ctx.ShouldBindWith(&task, binding.Form); err != nil {
			log.Printf("could not bind task %s", err.Error())
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := gorm.G[models.Task](db).Create(ctx, &task); err != nil {
			log.Printf("could not create task %s", err.Error())
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not create task"})
			return
		}

		FetchAndRenderTasks(ctx, db)

	})

	r.POST("/update-task", func(ctx *gin.Context) {
		id := ctx.PostForm("ID")
		if id == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing id"})
			return
		}

		// Only update the columns that were actually submitted, so a partial
		// form (e.g. just toggling Done) doesn't wipe the other fields. A map
		// also writes zero values ("" / false), which a struct Updates skips.
		updates := map[string]any{}
		if v, ok := ctx.GetPostForm("Name"); ok {
			updates["name"] = v
		}
		if v, ok := ctx.GetPostForm("Description"); ok {
			updates["description"] = v
		}
		if v, ok := ctx.GetPostForm("Done"); ok {
			updates["done"] = v == "true"
		}

		if len(updates) == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
			return
		}

		// Model(&Task{}) keeps gorm's soft-delete scope (deleted_at IS NULL) and
		// maps the snake_case column names; Updates(map) handles partial updates.
		res := db.Model(&models.Task{}).Where("id = ?", id).Updates(updates)

		if res.Error != nil {
			log.Printf("could not update task %s", res.Error.Error())
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not update task"})
			return
		}

		if res.RowsAffected == 0 {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		FetchAndRenderTasks(ctx, db)
	})

	r.POST("/delete-task", func(ctx *gin.Context) {
		id := ctx.Query("id")

		rows, err := gorm.G[models.Task](db).Where("id = ?", id).Delete(ctx)

		if err != nil {
			log.Printf("could not delete task %s", err.Error())
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete task"})
			return
		}

		if rows == 0 {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		FetchAndRenderTasks(ctx, db)

	})

	// Start server on port 8080 (default)
	// Server will listen on 0.0.0.0:8080 (localhost:8080 on Windows)
	r.Run()
}

func FetchAndRenderTasks(ctx *gin.Context, db *gorm.DB) {
	tasks, err := gorm.G[models.Task](db).Find(ctx)
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
