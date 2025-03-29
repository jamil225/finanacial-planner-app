package chat

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"time"

	"financial-planner-app/service"

	"github.com/openai/openai-go"
)

// ChatInterface handles the user interaction with the assistant
type ChatInterface struct {
	service   *service.OpenAIService
	thread    *openai.Thread
	assistant *openai.Assistant
}

// NewChatInterface creates a new chat interface
func NewChatInterface(service *service.OpenAIService, thread *openai.Thread, assistant *openai.Assistant) *ChatInterface {
	return &ChatInterface{
		service:   service,
		thread:    thread,
		assistant: assistant,
	}
}

// Start starts the chat interface
func (c *ChatInterface) Start() error {
	log.Printf("Ready to chat with the assistant")
	fmt.Println("Welcome to the Financial Assistant Chatbot!")
	fmt.Println("Type 'exit' to quit.")

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\nYou: ")
		userInput, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}

		if userInput == "exit\n" {
			fmt.Println("Goodbye!")
			break
		}

		response, err := c.service.SendMessage(c.thread, userInput, c.assistant)
		if err != nil {
			log.Printf("Error sending message: %v", err)
			continue
		}
		fmt.Printf("Assistant: %s\n", response)

		time.Sleep(1 * time.Second)
	}

	return nil
}
