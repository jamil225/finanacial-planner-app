package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAIServiceInterface defines the interface for OpenAI service operations
type OpenAIServiceInterface interface {
	CreateOrGetAssistant(assistantID string) (*openai.Assistant, error)
	CreateVectorStore() (string, error)
	AddVectorStoreToAssistant(assistant *openai.Assistant, vectorID string) (*openai.Assistant, error)
	CreateThread() (*openai.Thread, error)
	SendMessage(thread *openai.Thread, message string, assistant *openai.Assistant) (string, error)
	StreamMessage(thread *openai.Thread, content string, assistant *openai.Assistant, streamChan chan<- string) error
	Client() *openai.Client
	Context() context.Context
}

// OpenAIService handles all OpenAI API interactions
type OpenAIService struct {
	client *openai.Client
	ctx    context.Context
}

// Client returns the OpenAI client
func (s *OpenAIService) Client() *openai.Client {
	return s.client
}

// Context returns the context
func (s *OpenAIService) Context() context.Context {
	return s.ctx
}

// NewOpenAIService creates a new OpenAI service instance
func NewOpenAIService(apiKey string) *OpenAIService {
	if apiKey == "" {
		log.Fatal("OpenAI API key is required")
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &OpenAIService{
		client: client,
		ctx:    context.Background(),
	}
}

// CreateOrGetAssistant creates a new assistant or retrieves an existing one
func (s *OpenAIService) CreateOrGetAssistant(assistantID string) (*openai.Assistant, error) {
	list, err := s.client.Beta.Assistants.List(s.ctx, openai.BetaAssistantListParams{
		Order: openai.F(openai.BetaAssistantListParamsOrderDesc),
	})
	if err != nil {
		return nil, fmt.Errorf("error listing assistants: %w", err)
	}

	for _, assistant := range list.Data {
		if assistant.ID == assistantID {
			log.Printf("Found existing assistant: %s", assistant.ID)
			return &assistant, nil
		}
	}

	assistantPrompt := readFileContent("assistant_prompt.txt")
	assistant, err := s.client.Beta.Assistants.New(s.ctx, openai.BetaAssistantNewParams{
		Name:         openai.String("Financial Assistant"),
		Instructions: openai.String(assistantPrompt),
		Tools: openai.F([]openai.AssistantToolUnionParam{
			openai.FileSearchToolParam{Type: openai.F(openai.FileSearchToolTypeFileSearch)},
		}),
		Model: openai.String("gpt-4-1106-preview"),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating assistant: %w", err)
	}

	log.Printf("Created new assistant: %s", assistant.ID)
	return assistant, nil
}

// CreateVectorStore creates a new vector store and uploads files
func (s *OpenAIService) CreateVectorStore() (string, error) {
	vectorStore, err := s.client.Beta.VectorStores.New(
		s.ctx,
		openai.BetaVectorStoreNewParams{
			ExpiresAfter: openai.F(openai.BetaVectorStoreNewParamsExpiresAfter{
				Anchor: openai.F(openai.BetaVectorStoreNewParamsExpiresAfterAnchorLastActiveAt),
				Days:   openai.Int(1),
			}),
			Name: openai.String("Financial Statements"),
		},
	)
	if err != nil {
		return "", fmt.Errorf("error creating vector store: %w", err)
	}

	files := listFilesFromFolder("files")
	fileParams := make([]openai.FileNewParams, 0, len(files))

	for _, filePath := range files {
		rdr, err := os.Open(filePath)
		if err != nil {
			return "", fmt.Errorf("error opening file %s: %w", filePath, err)
		}
		defer rdr.Close()

		fileParams = append(fileParams, openai.FileNewParams{
			File:    openai.F[io.Reader](rdr),
			Purpose: openai.F(openai.FilePurposeAssistants),
		})
	}

	batch, err := s.client.Beta.VectorStores.FileBatches.UploadAndPoll(s.ctx, vectorStore.ID, fileParams, []string{}, 0)
	if err != nil {
		return "", fmt.Errorf("error uploading files: %w", err)
	}

	log.Printf("Created vector store %s with batch status: %s", vectorStore.ID, batch.Status)
	return vectorStore.ID, nil
}

// AddVectorStoreToAssistant adds a vector store to an assistant
func (s *OpenAIService) AddVectorStoreToAssistant(assistant *openai.Assistant, vectorID string) (*openai.Assistant, error) {
	updated, err := s.client.Beta.Assistants.Update(s.ctx, assistant.ID, openai.BetaAssistantUpdateParams{
		ToolResources: openai.F(openai.BetaAssistantUpdateParamsToolResources{
			FileSearch: openai.F(openai.BetaAssistantUpdateParamsToolResourcesFileSearch{
				VectorStoreIDs: openai.F([]string{vectorID}),
			}),
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("error updating assistant: %w", err)
	}

	return updated, nil
}

// CreateThread creates a new thread
func (s *OpenAIService) CreateThread() (*openai.Thread, error) {
	thread, err := s.client.Beta.Threads.New(s.ctx, openai.BetaThreadNewParams{})
	if err != nil {
		return nil, fmt.Errorf("error creating thread: %w", err)
	}
	return thread, nil
}

// SendMessage sends a message to the assistant and gets the response
func (s *OpenAIService) SendMessage(thread *openai.Thread, message string, assistant *openai.Assistant) (string, error) {
	_, err := s.client.Beta.Threads.Messages.New(s.ctx, thread.ID, openai.BetaThreadMessageNewParams{
		Content: openai.F([]openai.MessageContentPartParamUnion{
			openai.TextContentBlockParam{
				Text: openai.String(message),
				Type: openai.F(openai.TextContentBlockParamTypeText),
			},
		}),
		Role: openai.F(openai.BetaThreadMessageNewParamsRoleUser),
	})
	if err != nil {
		return "", fmt.Errorf("error creating message: %w", err)
	}

	threadPrompt := readFileContent("thread_prompt.txt")
	run, err := s.client.Beta.Threads.Runs.NewAndPoll(s.ctx, thread.ID, openai.BetaThreadRunNewParams{
		AssistantID:            openai.F(assistant.ID),
		AdditionalInstructions: openai.String(threadPrompt),
	}, 0)
	if err != nil {
		return "", fmt.Errorf("error running assistant: %w", err)
	}

	if run.Status == openai.RunStatusCompleted {
		messages, err := s.client.Beta.Threads.Messages.List(s.ctx, thread.ID, openai.BetaThreadMessageListParams{})
		if err != nil {
			return "", fmt.Errorf("error listing messages: %w", err)
		}

		// Get the last AI message
		for _, data := range messages.Data {
			if data.Role == "assistant" {
				for _, content := range data.Content {
					if content.Type == "text" {
						return content.Text.Value, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("no response from assistant")
}

// StreamMessage streams a message from the assistant
func (s *OpenAIService) StreamMessage(thread *openai.Thread, content string, assistant *openai.Assistant, streamChan chan<- string) error {
	// Create a new message
	_, err := s.client.Beta.Threads.Messages.New(s.ctx, thread.ID, openai.BetaThreadMessageNewParams{
		Content: openai.F([]openai.MessageContentPartParamUnion{
			openai.TextContentBlockParam{
				Text: openai.String(content),
				Type: openai.F(openai.TextContentBlockParamTypeText),
			},
		}),
		Role: openai.F(openai.BetaThreadMessageNewParamsRoleUser),
	})
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	// Create a run
	run, err := s.client.Beta.Threads.Runs.New(s.ctx, thread.ID, openai.BetaThreadRunNewParams{
		AssistantID: openai.F(assistant.ID),
	})
	if err != nil {
		return fmt.Errorf("failed to create run: %w", err)
	}

	// Poll for run completion and stream messages
	for {
		run, err = s.client.Beta.Threads.Runs.Get(s.ctx, thread.ID, run.ID)
		if err != nil {
			return fmt.Errorf("failed to get run: %w", err)
		}

		if run.Status == openai.RunStatusCompleted {
			// Get the latest message
			messages, err := s.client.Beta.Threads.Messages.List(s.ctx, thread.ID, openai.BetaThreadMessageListParams{
				Order: openai.F(openai.BetaThreadMessageListParamsOrderDesc),
				Limit: openai.F(int64(1)),
			})
			if err != nil {
				return fmt.Errorf("failed to list messages: %w", err)
			}

			if len(messages.Data) > 0 {
				// Stream the content character by character
				for _, content := range messages.Data[0].Content {
					if content.Type == "text" {
						for _, char := range content.Text.Value {
							streamChan <- string(char)
							time.Sleep(50 * time.Millisecond) // Add a small delay for natural typing effect
						}
					}
				}
			}
			return nil
		}

		if run.Status == openai.RunStatusFailed {
			return fmt.Errorf("run failed: %v", run.LastError)
		}

		time.Sleep(1 * time.Second)
	}
}

// Helper functions
func readFileContent(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading file %s: %v", filePath, err)
	}
	return string(content)
}

func listFilesFromFolder(folderPath string) []string {
	var files []string
	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error walking path %q: %v", folderPath, err)
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		log.Printf("Error walking path %q: %v", folderPath, err)
	}
	return files
}
