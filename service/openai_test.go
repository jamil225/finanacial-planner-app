package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/openai/openai-go"
	"github.com/stretchr/testify/assert"
)

// MockOpenAIClient is a mock of the OpenAI client
type MockOpenAIClient struct {
	ctrl *gomock.Controller
}

func (m *MockOpenAIClient) CreateOrGetAssistant(assistantID string) (*openai.Assistant, error) {
	return &openai.Assistant{ID: assistantID}, nil
}

func (m *MockOpenAIClient) CreateVectorStore() (string, error) {
	return "test-vector-store", nil
}

func (m *MockOpenAIClient) AddVectorStoreToAssistant(assistant *openai.Assistant, vectorID string) (*openai.Assistant, error) {
	return assistant, nil
}

func (m *MockOpenAIClient) CreateThread() (*openai.Thread, error) {
	return &openai.Thread{ID: "test-thread-id"}, nil
}

func (m *MockOpenAIClient) SendMessage(thread *openai.Thread, message string, assistant *openai.Assistant) (string, error) {
	return "Mock response", nil
}

func (m *MockOpenAIClient) StreamMessage(thread *openai.Thread, content string, assistant *openai.Assistant, streamChan chan<- string) error {
	return nil
}

func (m *MockOpenAIClient) Client() *openai.Client {
	return &openai.Client{}
}

func (m *MockOpenAIClient) Context() context.Context {
	return context.Background()
}

func TestNewOpenAIService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := &MockOpenAIClient{ctrl: ctrl}
	service := &OpenAIService{
		client: mockClient.Client(),
		ctx:    mockClient.Context(),
	}

	assert.NotNil(t, service)
	assert.NotNil(t, service.client)
	assert.NotNil(t, service.ctx)
}

func TestCreateOrGetAssistant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := &MockOpenAIClient{ctrl: ctrl}
	service := &OpenAIService{
		client: mockClient.Client(),
		ctx:    mockClient.Context(),
	}

	// Test case: Assistant exists
	t.Run("existing assistant", func(t *testing.T) {
		assistant, err := service.CreateOrGetAssistant("asst_v3GzI9KkkvrJTXWNn0w7Zfya")
		assert.NoError(t, err)
		assert.NotNil(t, assistant)
	})

	// Test case: Create new assistant
	t.Run("create new assistant", func(t *testing.T) {
		assistant, err := service.CreateOrGetAssistant("non-existent-id")
		assert.NoError(t, err)
		assert.NotNil(t, assistant)
	})
}

func TestCreateThread(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := &MockOpenAIClient{ctrl: ctrl}
	service := &OpenAIService{
		client: mockClient.Client(),
		ctx:    mockClient.Context(),
	}

	thread, err := service.CreateThread()
	assert.NoError(t, err)
	assert.NotNil(t, thread)
}

func TestSendMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := &MockOpenAIClient{ctrl: ctrl}
	service := &OpenAIService{
		client: mockClient.Client(),
		ctx:    mockClient.Context(),
	}

	// Create test thread and assistant
	thread := &openai.Thread{ID: "test-thread-id"}
	assistant := &openai.Assistant{ID: "test-assistant-id"}

	// Test sending a message
	response, err := service.SendMessage(thread, "Hello, how can you help me?", assistant)
	assert.NoError(t, err)
	assert.NotEmpty(t, response)
	assert.Equal(t, "Mock response", response)
}
