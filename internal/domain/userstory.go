package domain

type UserStory struct {
	ID          string // A unique identifier for the story
	Description string // The main text of the user story
	Category    string // Category assigned by the LLM (e.g., "Feature", "Bug", "Chore")
}
