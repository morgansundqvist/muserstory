package adapters

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/morgansundqvist/muserstory/internal/domain"
)

type JsonUserStoryRepository struct {
	mu       sync.Mutex
	filePath string
	projects map[string]domain.Project
	ticker   *time.Ticker
	doneChan chan bool
}

func NewJsonUserStoryRepository(filePath string) (*JsonUserStoryRepository, error) {
	repo := &JsonUserStoryRepository{
		filePath: filePath,
		projects: make(map[string]domain.Project),
		doneChan: make(chan bool),
	}

	if err := repo.loadFromFile(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error loading data from file: %w", err)
		}
	}

	repo.ticker = time.NewTicker(20 * time.Second)
	go repo.autoSave()

	return repo, nil
}

func (r *JsonUserStoryRepository) loadFromFile() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return err 
	}

	if len(data) == 0 { 
		r.projects = make(map[string]domain.Project)
		return nil
	}

	var projects []domain.Project
	if err := json.Unmarshal(data, &projects); err != nil {
		return fmt.Errorf("error unmarshalling data from file: %w", err)
	}

	r.projects = make(map[string]domain.Project)
	for _, p := range projects {
		r.projects[p.ID] = p
	}
	return nil
}

func (r *JsonUserStoryRepository) saveToFile() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var projectsList []domain.Project
	for _, p := range r.projects {
		projectsList = append(projectsList, p)
	}

	data, err := json.MarshalIndent(projectsList, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling projects to JSON: %w", err)
	}

	if err := os.WriteFile(r.filePath, data, 0644); err != nil {
		return fmt.Errorf("error writing data to file: %w", err)
	}
	return nil
}

func (r *JsonUserStoryRepository) autoSave() {
	for {
		select {
		case <-r.ticker.C:
			if err := r.saveToFile(); err != nil {
				fmt.Fprintf(os.Stderr, "Error auto-saving data: %v\\n", err)
			}
		case <-r.doneChan:
			r.ticker.Stop()
			return
		}
	}
}

func (r *JsonUserStoryRepository) StopAutoSave() {
	r.doneChan <- true
}

func (r *JsonUserStoryRepository) StoreProject(project domain.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if project.ID == "" {
		return fmt.Errorf("project ID cannot be empty")
	}
	r.projects[project.ID] = project
	return nil
}

func (r *JsonUserStoryRepository) GetProjects() ([]domain.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	projectsList := make([]domain.Project, 0, len(r.projects))
	for _, p := range r.projects {
		projectsList = append(projectsList, p)
	}
	return projectsList, nil
}

func (r *JsonUserStoryRepository) GetProjectByID(id string) (domain.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	project, ok := r.projects[id]
	if !ok {
		return domain.Project{}, fmt.Errorf("project with ID '%s' not found", id)
	}
	return project, nil
}
