package config

import (
	"log"
	"os"
)

// Config holds all configuration values
type Config struct {
	OpenAIAPIKey string
	AssistantID  string
	ModelName    string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {

	apiKey := os.Getenv("OPENAI_API_KEY_1")
	if apiKey == "" {
		return nil, ErrMissingAPIKey
	}
	log.Printf("API Key_1: %s", apiKey)

	assistantID := os.Getenv("ASSISTANT_ID")
	if assistantID == "" {
		assistantID = "asst_v3GzI9KkkvrJTXWNn0w7Zfya" // Default value
	}

	modelName := os.Getenv("MODEL_NAME")
	if modelName == "" {
		modelName = "gpt-4-1106-preview" // Default value
	}

	return &Config{
		OpenAIAPIKey: apiKey,
		AssistantID:  assistantID,
		ModelName:    modelName,
	}, nil
}
