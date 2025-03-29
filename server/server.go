package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"financial-planner-app/config"
	"financial-planner-app/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/openai/openai-go"
)

type Server struct {
	router        *gin.Engine
	openaiService *service.OpenAIService
	thread        *openai.Thread
	assistant     *openai.Assistant
	clients       map[*websocket.Conn]bool
	broadcast     chan Message
	mu            sync.Mutex
}

type Message struct {
	Type     string `json:"type"`
	Content  string `json:"content"`
	Sender   string `json:"sender"`
	IsStream bool   `json:"isStream"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

func NewServer() (*Server, error) {
	log.Println("Starting server initialization...")

	// Load configuration
	log.Println("Loading configuration...")
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("Failed to load configuration: %v", err)
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	log.Printf("Configuration loaded successfully. Assistant ID: %s", cfg.AssistantID)

	// Initialize OpenAI service
	log.Println("Initializing OpenAI service...")
	openaiService := service.NewOpenAIService(cfg.OpenAIAPIKey)
	log.Println("OpenAI service initialized successfully")

	// Create or get assistant
	log.Printf("Creating or getting assistant with ID: %s", cfg.AssistantID)
	assistant, err := openaiService.CreateOrGetAssistant(cfg.AssistantID)
	if err != nil {
		log.Printf("Failed to create/get assistant: %v", err)
		return nil, fmt.Errorf("failed to create/get assistant: %w", err)
	}
	log.Printf("Assistant created/retrieved successfully. ID: %s", assistant.ID)

	// Create a new thread
	log.Println("Creating new thread...")
	thread, err := openaiService.CreateThread()
	if err != nil {
		log.Printf("Failed to create thread: %v", err)
		return nil, fmt.Errorf("failed to create thread: %w", err)
	}
	log.Printf("Thread created successfully. ID: %s", thread.ID)

	// Initialize Gin
	log.Println("Initializing Gin router...")
	router := gin.Default()

	// Configure CORS
	log.Println("Configuring CORS...")
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
	log.Println("CORS configured successfully")

	// Create server instance
	log.Println("Creating server instance...")
	server := &Server{
		router:        router,
		openaiService: openaiService,
		thread:        thread,
		assistant:     assistant,
		clients:       make(map[*websocket.Conn]bool),
		broadcast:     make(chan Message),
	}
	log.Println("Server instance created successfully")

	// Setup routes
	log.Println("Setting up routes...")
	server.setupRoutes()
	log.Println("Routes configured successfully")

	// Start broadcasting messages
	log.Println("Starting message broadcast handler...")
	go server.handleMessages()
	log.Println("Message broadcast handler started successfully")

	log.Println("Server initialization completed successfully")
	return server, nil
}

func (s *Server) setupRoutes() {
	// API routes first
	api := s.router.Group("/api")
	{
		api.POST("/send", s.handleSendMessage)
		api.POST("/upload", s.handleFileUpload)
	}

	// WebSocket route
	s.router.GET("/ws", s.handleWebSocket)

	// Serve static files last
	s.router.NoRoute(gin.WrapH(http.FileServer(http.Dir("static"))))
}

func (s *Server) handleWebSocket(c *gin.Context) {
	log.Println("New WebSocket connection request...")
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Register client
	s.mu.Lock()
	s.clients[ws] = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.clients, ws)
		s.mu.Unlock()
		ws.Close()
		log.Println("Client disconnected")
	}()

	// Send welcome message
	welcomeMsg := Message{
		Type:    "system",
		Content: "Connected to Financial Assistant",
		Sender:  "system",
	}
	ws.WriteJSON(welcomeMsg)
	log.Println("Welcome message sent to client")

	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		log.Printf("Received message from client: %s", msg.Content)
		// Handle the message
		go s.handleWebSocketMessage(ws, msg)
	}
}

func (s *Server) handleWebSocketMessage(ws *websocket.Conn, msg Message) {
	log.Printf("Processing message: %s", msg.Content)

	// Create a channel for streaming responses
	streamChan := make(chan string)
	done := make(chan bool)

	// Start streaming in a goroutine
	go func() {
		defer close(done)
		err := s.openaiService.StreamMessage(s.thread, msg.Content, s.assistant, streamChan)
		if err != nil {
			log.Printf("Error streaming message: %v", err)
			ws.WriteJSON(Message{
				Type:    "error",
				Content: "Error processing message",
				Sender:  "system",
			})
			return
		}
	}()

	// Stream responses to the client
	for {
		select {
		case chunk, ok := <-streamChan:
			if !ok {
				return
			}
			err := ws.WriteJSON(Message{
				Type:     "ai",
				Content:  chunk,
				Sender:   "ai",
				IsStream: true,
			})
			if err != nil {
				log.Printf("Error sending stream chunk: %v", err)
				return
			}
		case <-done:
			// Send final message to indicate stream is complete
			ws.WriteJSON(Message{
				Type:     "ai",
				Content:  "",
				Sender:   "ai",
				IsStream: false,
			})
			return
		}
	}
}

func (s *Server) handleMessages() {
	for {
		msg := <-s.broadcast
		s.mu.Lock()
		for client := range s.clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("WebSocket write error: %v", err)
				client.Close()
				delete(s.clients, client)
			}
		}
		s.mu.Unlock()
	}
}

func (s *Server) handleSendMessage(c *gin.Context) {
	var request struct {
		Message string `json:"message"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get assistant
	assistant, err := s.openaiService.CreateOrGetAssistant("asst_v3GzI9KkkvrJTXWNn0w7Zfya")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Send message
	aiResponse, err := s.openaiService.SendMessage(s.thread, request.Message, assistant)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"response": aiResponse,
	})
}

func (s *Server) handleFileUpload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create uploads directory if it doesn't exist
	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Save file
	filename := filepath.Base(file.Filename)
	if err := c.SaveUploadedFile(file, filepath.Join(uploadDir, filename)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add file to vector store
	vectorID, err := s.openaiService.CreateVectorStore()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get assistant and update it with the new vector store
	assistant, err := s.openaiService.CreateOrGetAssistant("asst_v3GzI9KkkvrJTXWNn0w7Zfya")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_, err = s.openaiService.AddVectorStoreToAssistant(assistant, vectorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"file":   filename,
	})
}

func (s *Server) Run(addr string) error {
	log.Printf("Server starting on %s", addr)
	return s.router.Run(addr)
}
