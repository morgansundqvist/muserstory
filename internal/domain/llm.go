package domain

import (
	"github.com/invopop/jsonschema"
)

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
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}
