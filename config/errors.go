package config

import "errors"

var (
	// ErrMissingAPIKey is returned when the OpenAI API key is not set
	ErrMissingAPIKey = errors.New("OPENAI_API_KEY environment variable is not set")
)
