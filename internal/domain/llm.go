package domain

import (
	"github.com/invopop/jsonschema"
)

// ModelType defines the available LLM model types.
type ModelType string

const (
	ModelTypeSimple            ModelType = "Simple"
	ModelTypeAdvanced          ModelType = "Advanced"
	ModelTypeReasoningSimple   ModelType = "ReasoningSimple"
	ModelTypeReasoningAdvanced ModelType = "ReasoningAdvanced"
)

type LLMSimpleInput struct {
	SystemMessage string
	UserMessage   string
	ModelType     ModelType
}

type LLMAdvancedInput struct {
	SystemMessage     string
	UserMessage       string
	ModelType         ModelType
	SchemaName        string
	SchemaDescription string
	Schema            interface{}
}

func GenerateSchema[T any]() interface{} {
	// Structured Outputs uses a subset of JSON schema
	// These flags are necessary to comply with the subset
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}
