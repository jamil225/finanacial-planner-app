package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is not set")
	}

	ctx := context.Background()
	client := openai.NewClient(option.WithAPIKey(apiKey))

	//Step 1 crete or get Assistants with file search tool
	assistant := createNewAssistant(client, ctx) // list all  fist

	//Step -2 create a vector store and upload the files
	vectorId := createVectorStore(ctx, client) //TODO attach fiels  directly write about costing of tockens

	//Step 3 add files to assistant
	assistant = addVectorStoreToAssistant(assistant, client, ctx, vectorId)

	//Step 4 create a new thread.
	thread, err := client.Beta.Threads.New(ctx, openai.BetaThreadNewParams{})
	if err != nil {
		panic(err.Error())
	}

	log.Printf("Ready to chat with the assistant")
	// Read user input
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Welcome to the Finanacial Asstent Chatbot!")
	fmt.Println("Type 'exit' to quit.")
	for {
		// Get user input
		fmt.Print("\nYou: ")
		userInput, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Error reading input: %v", err)
		}

		// Exit condition
		if userInput == "exit\n" {
			fmt.Println("Goodbye!")
			break
		}
		// Step 4 - Send the user message to the assistant thread and get the response
		sendMessageAndGetResponse(client, ctx, thread, userInput, assistant)
		time.Sleep(1 * time.Second)
	}

}

func sendMessageAndGetResponse(client *openai.Client, ctx context.Context, thread *openai.Thread, useMessage string, assistant *openai.Assistant) {
	threadMessage, err := client.Beta.Threads.Messages.New(ctx, thread.ID, openai.BetaThreadMessageNewParams{
		Content: openai.F([]openai.MessageContentPartParamUnion{
			openai.TextContentBlockParam{
				Text: openai.String(useMessage),
				Type: openai.F(openai.TextContentBlockParamTypeText),
			},
		}),
		Role: openai.F(openai.BetaThreadMessageNewParamsRoleUser),
	})
	if err != nil {
		panic(err.Error())
	}
	log.Printf("Created a new thread Message with id %s", threadMessage.ID)

	// pollIntervalMs of 0 uses default polling interval.
	threadPrompt := readFileContent("thread_prompt.txt")
	run, err := client.Beta.Threads.Runs.NewAndPoll(ctx, thread.ID, openai.BetaThreadRunNewParams{
		AssistantID:            openai.F(assistant.ID),
		AdditionalInstructions: openai.String(threadPrompt),
	}, 0)

	if err != nil {
		panic(err.Error())
	}

	if run.Status == openai.RunStatusCompleted {
		messages, err := client.Beta.Threads.Messages.List(ctx, thread.ID, openai.BetaThreadMessageListParams{})

		if err != nil {
			panic(err.Error())
		}

		for _, data := range messages.Data {
			for _, content := range data.Content {
				println(content.Text.Value)
			}
		}
	}
}

func addVectorStoreToAssistant(assistant *openai.Assistant, client *openai.Client, ctx context.Context, vectorId string) *openai.Assistant {
	log.Printf("Adding the vector store Id: %s to the assistant Id: %s", vectorId, assistant.ID)
	assistant, _ = client.Beta.Assistants.Update(ctx, assistant.ID, openai.BetaAssistantUpdateParams{
		ToolResources: openai.F(openai.BetaAssistantUpdateParamsToolResources{
			FileSearch: openai.F(openai.BetaAssistantUpdateParamsToolResourcesFileSearch{
				VectorStoreIDs: openai.F([]string{vectorId}),
			}),
		}),
	})
	return assistant
}

func createNewAssistant(client *openai.Client, ctx context.Context) *openai.Assistant {
	log.Printf("Create or get  an assistant with file search tool")
	list, err := client.Beta.Assistants.List(ctx, openai.BetaAssistantListParams{
		Order: openai.F(openai.BetaAssistantListParamsOrderDesc),
	})
	if err != nil {
		log.Fatalf("Error listing assistants: %v", err)
	}

	// Loop over the list of assistants and print their details
	for _, assistant := range list.Data {
		if assistant.ID == "asst_v3GzI9KkkvrJTXWNn0w7Zfya" {
			fmt.Printf("Getting Assistant ID: %s, Name: %s\n", assistant.ID, assistant.Name)
			return &assistant
		}
	}
	assistantPrompt := readFileContent("assistant_prompt.txt")
	assistant, err := client.Beta.Assistants.New(ctx, openai.BetaAssistantNewParams{
		Name:         openai.String("Financial Assistant"),
		Instructions: openai.String(assistantPrompt),
		Tools: openai.F([]openai.AssistantToolUnionParam{
			openai.FileSearchToolParam{Type: openai.F(openai.FileSearchToolTypeFileSearch)},
		}),
		Model: openai.String("gpt-4-1106-preview"),
	})
	log.Printf("Created an assistant with id %s", assistant.ID)
	if err != nil {
		log.Fatalf("Error creating assistant: %v", err)
		panic(err.Error())
	}
	return assistant
}

func createVectorStore(ctx context.Context, client *openai.Client) string {
	log.Printf("Creating a vector store and uploading files")
	vectorStore, err := client.Beta.VectorStores.New(
		ctx,
		openai.BetaVectorStoreNewParams{
			ExpiresAfter: openai.F(openai.BetaVectorStoreNewParamsExpiresAfter{
				Anchor: openai.F(openai.BetaVectorStoreNewParamsExpiresAfterAnchorLastActiveAt),
				Days:   openai.Int(1),
			}),
			Name: openai.String("Financial Statements"),
		},
	)

	if err != nil {
		panic(err)
	}
	var fileParams []openai.FileNewParams
	folderPath := "files"
	files := listFilesFromFolder(folderPath)
	for _, filePath := range files {
		fmt.Println(filePath)
		rdr, err := os.Open(filePath)
		if err != nil {
			panic("file open failed:" + err.Error())
		}
		defer rdr.Close()

		fileParams = append(fileParams, openai.FileNewParams{
			File:    openai.F[io.Reader](rdr),
			Purpose: openai.F(openai.FilePurposeAssistants),
		})
	}

	// 0 uses default polling interval
	batch, err := client.Beta.VectorStores.FileBatches.UploadAndPoll(ctx, vectorStore.ID, fileParams,
		[]string{}, 0)

	if err != nil {
		panic(err)
	}
	println("batchStatus : " + batch.Status)

	println("Listing the files from the vector store" + vectorStore.ID)

	filesObjects, err := client.Beta.VectorStores.Files.List(ctx, vectorStore.ID, openai.BetaVectorStoreFileListParams{})
	if err != nil {
		panic(err)
	}
	println("Files in the vector store : ")
	for _, file := range filesObjects.Data {
		println(file.ID)
	}
	log.Printf("Created a vector store with id %s", vectorStore.ID)
	return vectorStore.ID
}

func listFilesFromFolder(folderPath string) []string {
	log.Println("Listing files from the folder: ", folderPath)
	var files []string
	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error walking the path %q: %v\n", folderPath, err)
		}
		if !info.IsDir() {
			fmt.Println(path)
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		log.Printf("Error walking the path %q: %v\n", folderPath, err)
	}
	log.Printf("Found %d files in the folder %s", len(files), folderPath)
	return files
}

// readFileContent reads the content of a file and returns it as a string
func readFileContent(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading the file %s", filePath)
	}
	return string(content)
}
