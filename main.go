package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "github.com/mattn/go-sqlite3"
)

type Task struct {
	gorm.Model
	Name        string
	Description string
	Done        bool
}

func main() {

	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})

	if err != nil {
		log.Printf("could not start db %s", err.Error())
		return
	}

	db.AutoMigrate(&Task{})

	// Create a Gin router with default middleware (logger and recovery)
	r := gin.Default()

	r.GET("/tasks", func(ctx *gin.Context) {
		tasks, err := gorm.G[Task](db).Find(ctx)

		if err != nil {
			log.Printf("could not get tasks %s", err.Error())
			return
		}

		retval, err := json.Marshal(tasks)

		if err != nil {
			log.Printf("marshall tasks %s", err.Error())
			return
		}

		ctx.JSON(http.StatusOK, string(retval))
	})

	r.GET("/task", func(ctx *gin.Context) {
		id := ctx.Query("id")

		task, err := gorm.G[Task](db).Where("id = ?", id).First(ctx)

		if err != nil {
			log.Printf("could not get task %s", err.Error())
			ctx.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		ctx.JSON(http.StatusOK, task)
	})

	r.POST("/add-task", func(ctx *gin.Context) {
		var task Task

		if err := ctx.ShouldBindJSON(&task); err != nil {
			log.Printf("could not bind task %s", err.Error())
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := gorm.G[Task](db).Create(ctx, &task); err != nil {
			log.Printf("could not create task %s", err.Error())
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not create task"})
			return
		}

		ctx.JSON(http.StatusCreated, task)
	})

	r.POST("/update-task", func(ctx *gin.Context) {
		var task Task

		if err := ctx.ShouldBindJSON(&task); err != nil {
			log.Printf("could not bind task %s", err.Error())
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		rows, err := gorm.G[Task](db).Where("id = ?", task.ID).Updates(ctx, task)

		if err != nil {
			log.Printf("could not update task %s", err.Error())
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not update task"})
			return
		}

		if rows == 0 {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		ctx.JSON(http.StatusOK, task)
	})

	r.POST("/delete-task", func(ctx *gin.Context) {
		id := ctx.Query("id")

		rows, err := gorm.G[Task](db).Where("id = ?", id).Delete(ctx)

		if err != nil {
			log.Printf("could not delete task %s", err.Error())
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete task"})
			return
		}

		if rows == 0 {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"deleted": id})
	})

	// Start server on port 8080 (default)
	// Server will listen on 0.0.0.0:8080 (localhost:8080 on Windows)
	r.Run()
}
