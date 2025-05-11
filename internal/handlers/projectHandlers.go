package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/morgansundqvist/muserstory/internal/domain"
	"github.com/morgansundqvist/muserstory/internal/ports"
)

type ProjectHandler struct {
	Repo ports.UserStoryRepository
}

func NewProjectHandler(repo ports.UserStoryRepository) *ProjectHandler {
	return &ProjectHandler{Repo: repo}
}

func (h *ProjectHandler) CreateProject(c *fiber.Ctx) error {
	project := new(domain.Project)
	if err := c.BodyParser(project); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "cannot parse JSON",
			"details": err.Error(),
		})
	}
	if project.ID == "" {
	}
	if err := h.Repo.StoreProject(*project); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "failed to store project",
			"details": err.Error(),
		})
	}
	return c.Status(fiber.StatusCreated).JSON(project)
}

func (h *ProjectHandler) GetProjects(c *fiber.Ctx) error {
	projects, err := h.Repo.GetProjects()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "failed to get projects",
			"details": err.Error(),
		})
	}
	return c.JSON(projects)
}

func (h *ProjectHandler) GetProjectByID(c *fiber.Ctx) error {
	id := c.Params("id")
	project, err := h.Repo.GetProjectByID(id)
	if err != nil {
		if err.Error() == "project with ID '"+id+"' not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "project not found",
				"details": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "failed to get project by ID",
			"details": err.Error(),
		})
	}
	return c.JSON(project)
}
