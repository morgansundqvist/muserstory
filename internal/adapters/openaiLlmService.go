package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/morgansundqvist/muserstory/internal/domain"
	"github.com/openai/openai-go"
)

type OpenAILLMService struct {
}

func NewOpenAILLMService() *OpenAILLMService {
	return &OpenAILLMService{}
}

func (s *OpenAILLMService) CategorizeStory(storyText string) (string, error) {
	fmt.Printf("MockLLMService: Attempting to categorize story: \"%s\"\n", storyText)
	if strings.Contains(strings.ToLower(storyText), "bug") || strings.Contains(strings.ToLower(storyText), "fix") {
		return "Bug", nil
	}
	if strings.Contains(strings.ToLower(storyText), "feature") || strings.Contains(strings.ToLower(storyText), "i want") {
		return "Feature", nil
	}
	if len(storyText)%3 == 0 { 
		return "Chore", nil
	}
	return "Technical Debt", nil 
}

func (s *OpenAILLMService) CategorizeStories(stories []domain.UserStory) ([]domain.UserStory, error) {
	fmt.Printf("MockLLMService: Attempting to categorize %d stories.\n", len(stories))
	updatedStories := make([]domain.UserStory, len(stories))
	for i, story := range stories {
		category, err := s.CategorizeStory(story.Description)
		if err != nil {
			fmt.Printf("Error categorizing story ID %s ('%s'): %v. Assigning 'Uncategorized'.\n", story.ID, story.Description, err)
			updatedStories[i] = story
			updatedStories[i].Category = "Uncategorized"
			continue
		}
		updatedStories[i] = story
		updatedStories[i].Category = category
	}
	return updatedStories, nil
}

func (s *OpenAILLMService) AskSimple(input domain.LLMSimpleInput) (string, error) {
	client := openai.NewClient()

	model := openai.ChatModelGPT4o

	if input.ModelType == domain.ModelTypeSimple {
		model = openai.ChatModelGPT4oMini
	} else if input.ModelType == domain.ModelTypeAdvanced {
		model = openai.ChatModelGPT4o
	} else if input.ModelType == domain.ModelTypeReasoningSimple {
		model = openai.ChatModelO3Mini
	} else if input.ModelType == domain.ModelTypeReasoningAdvanced {
		model = openai.ChatModelO1
	}

	chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(input.SystemMessage),
			openai.UserMessage(input.UserMessage),
		},
		Model: model,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get chat completion: %w", err)
	}
	println(chatCompletion.Choices[0].Message.Content)

	return chatCompletion.Choices[0].Message.Content, nil
}

func (s *OpenAILLMService) AskAdvanced(input domain.LLMAdvancedInput) (string, error) {

	client := openai.NewClient()

	model := openai.ChatModelGPT4o

	if input.ModelType == domain.ModelTypeSimple {
		model = openai.ChatModelGPT4oMini
	} else if input.ModelType == domain.ModelTypeAdvanced {
		model = openai.ChatModelGPT4o
	} else if input.ModelType == domain.ModelTypeReasoningSimple {
		model = openai.ChatModelO3Mini
	} else if input.ModelType == domain.ModelTypeReasoningAdvanced {
		model = openai.ChatModelO1
	}

	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        input.SchemaName,
		Description: openai.String(input.SchemaDescription),
		Schema:      input.Schema,
		Strict:      openai.Bool(true),
	}

	chat, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(input.SystemMessage),
			openai.UserMessage(input.UserMessage),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
				JSONSchema: schemaParam,
			},
		},
		Model: model,
	})

	if err != nil {
		return "", fmt.Errorf("failed to get chat completion: %w", err)
	}

	return chat.Choices[0].Message.Content, nil
}
