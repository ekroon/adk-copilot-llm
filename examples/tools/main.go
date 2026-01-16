// Package main demonstrates the usage of tool support in the adk-copilot-llm package.
//
// This example shows:
// - Defining a calculator tool using genai.Tool with FunctionDeclarations
// - Supporting multiple operations: add, subtract, multiply, divide
// - Creating a tool handler function that performs calculations
// - Registering the handler in copilot.Config.ToolHandlers
// - Making a request that triggers the tool
// - Displaying the LLM's response with the calculated result
//
// Prerequisites:
// - The Copilot CLI must be installed and available in PATH (or set COPILOT_CLI_PATH)
// - You must be authenticated with GitHub Copilot (the CLI handles this automatically)
//
// Note: Tool support in adk-copilot-llm is currently in development. This example
// demonstrates the intended API design and may require updates when full tool
// support is implemented in the copilot package.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ekroon/adk-copilot-llm/copilot"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

func main() {
	ctx := context.Background()

	// =========================================================================
	// Define the calculator tool
	// =========================================================================
	// Tools are defined using genai.Tool with FunctionDeclarations that
	// describe the available functions, their parameters, and their purpose.
	calculatorTool := &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "calculator",
				Description: "Performs basic arithmetic operations on two numbers",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"operation": {
							Type:        genai.TypeString,
							Description: "The arithmetic operation to perform",
							Enum:        []string{"add", "subtract", "multiply", "divide"},
						},
						"a": {
							Type:        genai.TypeNumber,
							Description: "The first number",
						},
						"b": {
							Type:        genai.TypeNumber,
							Description: "The second number",
						},
					},
					Required: []string{"operation", "a", "b"},
				},
			},
		},
	}

	// =========================================================================
	// Create the calculator tool handler
	// =========================================================================
	// The handler is invoked when the LLM calls the calculator tool.
	// It receives the arguments as a map and returns the result as a string.
	calculatorHandler := func(args map[string]any) (string, error) {
		// Extract operation parameter
		operation, ok := args["operation"].(string)
		if !ok {
			return "", fmt.Errorf("invalid or missing 'operation' parameter")
		}

		// Extract numeric parameters with type assertion
		// Numbers from JSON are typically float64
		aVal, ok := args["a"].(float64)
		if !ok {
			return "", fmt.Errorf("invalid or missing 'a' parameter")
		}

		bVal, ok := args["b"].(float64)
		if !ok {
			return "", fmt.Errorf("invalid or missing 'b' parameter")
		}

		// Perform the calculation based on the operation
		var result float64
		switch operation {
		case "add":
			result = aVal + bVal
		case "subtract":
			result = aVal - bVal
		case "multiply":
			result = aVal * bVal
		case "divide":
			// Handle division by zero
			if bVal == 0 {
				return "", fmt.Errorf("division by zero is not allowed")
			}
			result = aVal / bVal
		default:
			return "", fmt.Errorf("unsupported operation: %s", operation)
		}

		// Return the result in a clear format
		return fmt.Sprintf("%g %s %g = %g", aVal, operation, bVal, result), nil
	}

	// =========================================================================
	// Create the CopilotLLM instance with tool handler
	// =========================================================================
	// Register the calculator handler so it can be invoked when the LLM
	// decides to use the calculator tool.
	llm, err := copilot.New(copilot.Config{
		Model: "gpt-4",
		ToolHandlers: map[string]copilot.ToolHandler{
			"calculator": calculatorHandler,
		},
	})
	if err != nil {
		log.Fatalf("Failed to create Copilot LLM: %v", err)
	}
	defer llm.Close()

	fmt.Printf("Using LLM: %s\n\n", llm.Name())

	// =========================================================================
	// Example 1: Basic calculation request
	// =========================================================================
	// Ask the LLM a question that should trigger the calculator tool.
	// The LLM should recognize that it needs to use the calculator to compute
	// the result and then include that result in its response.
	fmt.Println("Example 1: Basic calculation")
	fmt.Println("============================")
	fmt.Println("Question: What is 42 multiplied by 17?")
	fmt.Println()

	request := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("What is 42 multiplied by 17?")},
			},
		},
		Config: &genai.GenerateContentConfig{
			Tools: []*genai.Tool{calculatorTool},
		},
	}

	// Process the response
	fmt.Print("Response: ")
	for resp, err := range llm.GenerateContent(ctx, request, false) {
		if err != nil {
			log.Fatalf("Error generating content: %v", err)
		}
		if resp.Content != nil && len(resp.Content.Parts) > 0 {
			for _, part := range resp.Content.Parts {
				fmt.Print(part.Text)
			}
		}
	}
	fmt.Println()

	// =========================================================================
	// Example 2: Division with error handling
	// =========================================================================
	// Demonstrate how the tool handler handles edge cases like division by zero.
	fmt.Println("Example 2: Division calculation")
	fmt.Println("================================")
	fmt.Println("Question: What is 100 divided by 5?")
	fmt.Println()

	divRequest := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("What is 100 divided by 5?")},
			},
		},
		Config: &genai.GenerateContentConfig{
			Tools: []*genai.Tool{calculatorTool},
		},
	}

	fmt.Print("Response: ")
	for resp, err := range llm.GenerateContent(ctx, divRequest, false) {
		if err != nil {
			log.Fatalf("Error generating content: %v", err)
		}
		if resp.Content != nil && len(resp.Content.Parts) > 0 {
			for _, part := range resp.Content.Parts {
				fmt.Print(part.Text)
			}
		}
	}
	fmt.Println()

	// =========================================================================
	// Example 3: Complex calculation with multiple operations
	// =========================================================================
	// The LLM may need to perform multiple calculations to answer the question.
	fmt.Println("Example 3: Complex calculation")
	fmt.Println("===============================")
	fmt.Println("Question: If I have 15 apples and add 7 more, then divide them equally among 2 people, how many does each person get?")
	fmt.Println()

	complexRequest := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("If I have 15 apples and add 7 more, then divide them equally among 2 people, how many does each person get?")},
			},
		},
		Config: &genai.GenerateContentConfig{
			Tools: []*genai.Tool{calculatorTool},
		},
	}

	fmt.Print("Response: ")
	for resp, err := range llm.GenerateContent(ctx, complexRequest, false) {
		if err != nil {
			log.Fatalf("Error generating content: %v", err)
		}
		if resp.Content != nil && len(resp.Content.Parts) > 0 {
			for _, part := range resp.Content.Parts {
				fmt.Print(part.Text)
			}
		}
	}
	fmt.Println()

	// =========================================================================
	// Example 4: Streaming with tools
	// =========================================================================
	// Demonstrate streaming responses when tools are involved.
	fmt.Println("Example 4: Streaming calculation")
	fmt.Println("=================================")
	fmt.Println("Question: Calculate 256 minus 128")
	fmt.Println()

	streamRequest := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("Calculate 256 minus 128")},
			},
		},
		Config: &genai.GenerateContentConfig{
			Tools: []*genai.Tool{calculatorTool},
		},
	}

	fmt.Print("Response: ")
	for resp, err := range llm.GenerateContent(ctx, streamRequest, true) {
		if err != nil {
			log.Fatalf("Error generating content: %v", err)
		}
		if resp.Content != nil && len(resp.Content.Parts) > 0 {
			for _, part := range resp.Content.Parts {
				fmt.Print(part.Text)
			}
		}
		if resp.TurnComplete {
			break
		}
	}
	fmt.Println()

	fmt.Println("All examples completed successfully!")
	fmt.Println()
	fmt.Println("Note: Tool support requires the copilot package to handle tool calls")
	fmt.Println("and invoke the registered handlers. If you see responses without")
	fmt.Println("calculated results, the tool integration may need further implementation.")
}
