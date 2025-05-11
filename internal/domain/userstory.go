package domain

type Project struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Summary     string      `json:"summary"`
	UserStories []UserStory `json:"user_stories"`
}

type UserStory struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Category    string `json:"category"`
}
