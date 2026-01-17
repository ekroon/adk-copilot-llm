// Package main demonstrates the usage of tool support in the adk-copilot-llm package
// using the new ADK tool.Tool interface with functiontool.New.
//
// This example shows:
// - Defining tools using functiontool.New with typed input/output structs
// - Using JSON schema tags for automatic schema generation
// - Supporting multiple operations: add, subtract, multiply, divide
// - Error handling (e.g., division by zero)
// - Multiple calculation examples with both streaming and non-streaming requests
//
// Prerequisites:
// - The Copilot CLI must be installed and available in PATH (or set COPILOT_CLI_PATH)
// - You must be authenticated with GitHub Copilot (the CLI handles this automatically)
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ekroon/adk-copilot-llm/copilot"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
)

// CalculatorInput defines the input parameters for the calculator tool.
type CalculatorInput struct {
	Operation string  `json:"operation" jsonschema:"enum=add,enum=subtract,enum=multiply,enum=divide,description=The arithmetic operation to perform"`
	A         float64 `json:"a" jsonschema:"description=The first number"`
	B         float64 `json:"b" jsonschema:"description=The second number"`
}

// CalculatorOutput defines the output of the calculator tool.
type CalculatorOutput struct {
	Result float64 `json:"result" jsonschema:"description=The result of the operation"`
}

// calculatorFunc is the handler function that performs the actual calculation.
func calculatorFunc(ctx tool.Context, input CalculatorInput) (CalculatorOutput, error) {
	var result float64

	switch input.Operation {
	case "add":
		result = input.A + input.B
	case "subtract":
		result = input.A - input.B
	case "multiply":
		result = input.A * input.B
	case "divide":
		// Handle division by zero
		if input.B == 0 {
			return CalculatorOutput{}, fmt.Errorf("division by zero is not allowed")
		}
		result = input.A / input.B
	default:
		return CalculatorOutput{}, fmt.Errorf("unsupported operation: %s", input.Operation)
	}

	return CalculatorOutput{Result: result}, nil
}

func main() {
	ctx := context.Background()

	// =========================================================================
	// Create the calculator tool using functiontool.New
	// =========================================================================
	// The functiontool.New API automatically generates JSON schemas from the
	// typed input and output structs. The jsonschema tags provide additional
	// metadata like descriptions and enum values.
	calculatorTool, err := functiontool.New(
		functiontool.Config{
			Name:        "calculator",
			Description: "Performs basic arithmetic operations (add, subtract, multiply, divide) on two numbers",
		},
		calculatorFunc,
	)
	if err != nil {
		log.Fatalf("Failed to create calculator tool: %v", err)
	}

	// =========================================================================
	// Create the CopilotLLM instance with the tool
	// =========================================================================
	// The calculator tool is passed to the copilot LLM so it can be used
	// when generating content.
	llm, err := copilot.New(copilot.Config{
		Model: "gpt-4",
		Tools: []tool.Tool{calculatorTool},
	})
	if err != nil {
		log.Fatalf("Failed to create Copilot LLM: %v", err)
	}
	defer llm.Close()

	fmt.Printf("Using LLM: %s\n\n", llm.Name())

	// =========================================================================
	// Example 1: Simple multiplication
	// =========================================================================
	// Ask the LLM a question that should trigger the calculator tool.
	// The LLM should recognize that it needs to use the calculator to compute
	// the result and then include that result in its response.
	fmt.Println("Example 1: Simple multiplication")
	fmt.Println("=================================")
	fmt.Println("Question: What is 42 multiplied by 17?")
	fmt.Println()

	request := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("What is 42 multiplied by 17?")},
			},
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
	fmt.Println()

	// =========================================================================
	// Example 3: Addition with complex phrasing
	// =========================================================================
	// The LLM should extract the numbers and operation from natural language.
	fmt.Println("Example 3: Addition calculation")
	fmt.Println("================================")
	fmt.Println("Question: If I have 15 apples and add 7 more, how many do I have?")
	fmt.Println()

	addRequest := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("If I have 15 apples and add 7 more, how many do I have?")},
			},
		},
	}

	fmt.Print("Response: ")
	for resp, err := range llm.GenerateContent(ctx, addRequest, false) {
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
	fmt.Println()

	// =========================================================================
	// Example 4: Streaming with tools
	// =========================================================================
	// Demonstrate streaming responses when tools are involved.
	// The response will arrive in chunks as it's being generated.
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
	fmt.Println()

	fmt.Println("All examples completed successfully!")
	fmt.Println()
	fmt.Println("Note: This example demonstrates the new tool.Tool interface using functiontool.New,")
	fmt.Println("which provides type-safe tool definitions with automatic schema generation from")
	fmt.Println("struct tags. The copilot package handles tool calls and invokes the registered tools.")
}
