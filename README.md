# Financial Assistant with OpenAI's Assistants API

A smart financial assistant chatbot built using OpenAI's Assistants API that helps users analyze financial documents and provide intelligent financial advice.

## Overview

This project implements a financial planning assistant that can:

- Analyze financial documents
- Provide personalized financial advice
- Answer questions about financial planning and budgeting
- Help users understand complex financial concepts
- Maintain context throughout conversations

## Features

- Interactive chat interface
- Document analysis capabilities
- Vector store integration for efficient document search
- Context-aware responses
- Professional financial guidance

## Prerequisites

- Go 1.22.5 or higher
- OpenAI API key
- Financial documents (sample documents provided)

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/financial-planner-app.git
   cd financial-planner-app
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Create a `files` directory and add your financial documents:
   ```bash
   mkdir files
   # Add your financial documents to the files directory
   ```

4. Update the API key in `main.go`:
   ```go
   client := openai.NewClient(option.WithAPIKey("your-api-key-here"))
   ```

## Usage

1. Run the application:
   ```bash
   go run main.go
   ```

2. Interact with the assistant through the command-line interface
3. Type your questions about financial planning
4. Type 'exit' to quit the application

## Project Structure

- `main.go` - Main application code
- `files/` - Directory containing financial documents
- `assistant_prompt.txt` - Instructions for the financial assistant
- `thread_prompt.txt` - Thread-specific instructions

## Tutorial

For a detailed tutorial on how this project was built, check out the article:
[Building a Smart Financial Assistant with OpenAI's Assistants API](https://medium.com/@jamil.ahmad7720/building-a-smart-financial-assistants-api-d6d3a8ec720c)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.


## Acknowledgments

- OpenAI for providing the Assistants API
- The Go community for excellent tools and libraries 