package ports

import "github.com/morgansundqvist/muserstory/internal/domain"

type UserStoryRepository interface {
	StoreProject(project domain.Project) error
	GetProjects() ([]domain.Project, error)
	GetProjectByID(id string) (domain.Project, error)
}
