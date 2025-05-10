package ports

import "github.com/morgansundqvist/muserstory/internal/domain"

type LLMService interface {
	AskSimple(input domain.LLMSimpleInput) (string, error)

	AskAdvanced(input domain.LLMAdvancedInput) (string, error)
}
