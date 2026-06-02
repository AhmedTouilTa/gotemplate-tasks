package main

import (
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
		ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(generator.Tasks(tasks)))
	})

	r.GET("/task", func(ctx *gin.Context) {
		id := ctx.Query("id")

		_, err := gorm.G[models.Task](db).Where("id = ?", id).First(ctx)

		if err != nil {
			log.Printf("could not get task %s", err.Error())
			ctx.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		tasks, err := gorm.G[models.Task](db).Find(ctx)
		if err != nil {
			log.Printf("could not get tasks %s", err.Error())
			return
		}
		ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(generator.Tasks(tasks)))
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

		tasks, err := gorm.G[models.Task](db).Find(ctx)
		if err != nil {
			log.Printf("could not get tasks %s", err.Error())
			return
		}
		ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(generator.Tasks(tasks)))
	})

	r.POST("/update-task", func(ctx *gin.Context) {
		var task models.Task

		if err := ctx.ShouldBindWith(&task, binding.Form); err != nil {
			log.Printf("could not bind task %s", err.Error())
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		rows, err := gorm.G[models.Task](db).Where("id = ?", task.ID).Updates(ctx, task)

		if err != nil {
			log.Printf("could not update task %s", err.Error())
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not update task"})
			return
		}

		if rows == 0 {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		tasks, err := gorm.G[models.Task](db).Find(ctx)
		if err != nil {
			log.Printf("could not get tasks %s", err.Error())
			return
		}
		ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(generator.Tasks(tasks)))
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

		tasks, err := gorm.G[models.Task](db).Find(ctx)
		if err != nil {
			log.Printf("could not get tasks %s", err.Error())
			return
		}
		ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(generator.Tasks(tasks)))
	})

	// Start server on port 8080 (default)
	// Server will listen on 0.0.0.0:8080 (localhost:8080 on Windows)
	r.Run()
}
