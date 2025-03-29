package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createTestEnvFile() error {
	content := []byte(`OPENAI_API_KEY=test-api-key
ASSISTANT_ID=asst_v3GzI9KkkvrJTXWNn0w7Zfya
MODEL_NAME=gpt-4-1106-preview`)
	return os.WriteFile(".env", content, 0644)
}

func TestLoadConfig(t *testing.T) {
	// Create test .env file
	err := createTestEnvFile()
	if err != nil {
		t.Fatalf("Failed to create test .env file: %v", err)
	}
	defer os.Remove(".env") // Clean up after test

	// Save original env and restore after test
	originalAPIKey := os.Getenv("OPENAI_API_KEY")
	defer os.Setenv("OPENAI_API_KEY", originalAPIKey)

	tests := []struct {
		name      string
		apiKey    string
		wantError bool
	}{
		{
			name:      "valid config",
			apiKey:    "test-api-key",
			wantError: false,
		},
		{
			name:      "missing api key",
			apiKey:    "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("OPENAI_API_KEY", tt.apiKey)

			cfg, err := LoadConfig()
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, cfg)
				assert.Equal(t, ErrMissingAPIKey, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, cfg)
			assert.Equal(t, tt.apiKey, cfg.OpenAIAPIKey)
			assert.Equal(t, "asst_v3GzI9KkkvrJTXWNn0w7Zfya", cfg.AssistantID)
			assert.Equal(t, "gpt-4-1106-preview", cfg.ModelName)
		})
	}
}
