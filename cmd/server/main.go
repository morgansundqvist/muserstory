package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/morgansundqvist/muserstory/internal/adapters"
	"github.com/morgansundqvist/muserstory/internal/handlers"
)


const dataFilePath = "projects.json" 

func main() {
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	repo, err := adapters.NewJsonUserStoryRepository(dataFilePath)
	if err != nil {
		log.Fatalf("Failed to initialize repository: %v", err)
	}
	defer repo.StopAutoSave() 

	app := fiber.New()

	app.Use(logger.New())

	api := app.Group("/api") 

	projectHandler := handlers.NewProjectHandler(repo)

	api.Post("/projects", projectHandler.CreateProject)
	api.Get("/projects", projectHandler.GetProjects)
	api.Get("/projects/:id", projectHandler.GetProjectByID)

	log.Printf("Starting server on http://localhost:%s\n", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
