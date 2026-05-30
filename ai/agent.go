package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// ─────────────────────────────────────────
//  Gemini API types
// ─────────────────────────────────────────

type GeminiRequest struct {
	Contents []GeminiContent        `json:"contents"`
	Tools    []GeminiToolDefinition `json:"tools"`
}

type GeminiContent struct {
	Role  string       `json:"role"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text         string              `json:"text,omitempty"`
	FunctionCall *GeminiFunctionCall `json:"functionCall,omitempty"`
	FunctionResp *GeminiFunctionResp `json:"functionResponse,omitempty"`
}

type GeminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type GeminiFunctionResp struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

type GeminiToolDefinition struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"functionDeclarations"`
}

type GeminiFunctionDeclaration struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Parameters  *GeminiParamSpec `json:"parameters,omitempty"`
}

type GeminiParamSpec struct {
	Type       string                    `json:"type"`
	Properties map[string]GeminiProperty `json:"properties,omitempty"`
	Required   []string                  `json:"required,omitempty"`
}

type GeminiProperty struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text         string `json:"text"`
				FunctionCall *struct {
					Name string                 `json:"name"`
					Args map[string]interface{} `json:"args"`
				} `json:"functionCall"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
}

// ─────────────────────────────────────────
//  MCP Tools — what the AI can call
// ─────────────────────────────────────────

var geminiTools = []GeminiToolDefinition{
	{
		FunctionDeclarations: []GeminiFunctionDeclaration{
			{
				Name:        "list_all_products",
				Description: "Returns all products in the ATX Technology inventory with stock levels, prices and categories.",
			},
			{
				Name:        "list_low_stock",
				Description: "Returns all products whose stock is at or below their low stock threshold. Use when asked about products needing restocking.",
			},
			{
				Name:        "get_product",
				Description: "Returns details for a single product by its ID.",
				Parameters: &GeminiParamSpec{
					Type: "object",
					Properties: map[string]GeminiProperty{
						"id": {Type: "string", Description: "The product ID e.g. ATX-001"},
					},
					Required: []string{"id"},
				},
			},
			{
				Name:        "get_inventory_summary",
				Description: "Returns a summary: total products, total stock value, count by category, and how many products are low on stock.",
			},
		},
	},
}

// ─────────────────────────────────────────
//  Product types + HTTP fetch
// ─────────────────────────────────────────

type Product struct {
	Id                string  `json:"id"`
	Name              string  `json:"name"`
	Description       string  `json:"description"`
	Category          int     `json:"category"`
	Price             float64 `json:"price"`
	Stock             int     `json:"stock"`
	Unit              int     `json:"unit"`
	LowStockThreshold int     `json:"low_stock_threshold"`
}

func fetchProducts() ([]Product, error) {
	resp, err := http.Get("http://localhost:8080/products")
	if err != nil {
		return nil, fmt.Errorf("cannot reach inventory server — is it running?")
	}
	defer resp.Body.Close()
	var products []Product
	json.NewDecoder(resp.Body).Decode(&products)
	return products, nil
}

func categoryName(c int) string {
	switch c {
	case 1:
		return "Fiber"
	case 2:
		return "LAN"
	case 3:
		return "Routers"
	case 4:
		return "Switches"
	case 5:
		return "Connectors"
	default:
		return "Unknown"
	}
}

func unitName(u int) string {
	switch u {
	case 1:
		return "metres"
	case 2:
		return "pieces"
	case 3:
		return "boxes"
	case 4:
		return "rolls"
	default:
		return "units"
	}
}

// ─────────────────────────────────────────
//  Tool execution
// ─────────────────────────────────────────

func executeTool(name string, args map[string]interface{}) string {
	products, err := fetchProducts()
	if err != nil {
		return err.Error()
	}

	switch name {

	case "list_all_products":
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("ATX Technology Inventory (%d products):\n\n", len(products)))
		for _, p := range products {
			sb.WriteString(fmt.Sprintf("• [%s] %s\n", p.Id, p.Name))
			sb.WriteString(fmt.Sprintf("  Category: %s | Price: $%.2f | Stock: %d %s\n",
				categoryName(p.Category), p.Price, p.Stock, unitName(p.Unit)))
		}
		return sb.String()

	case "list_low_stock":
		var low []Product
		for _, p := range products {
			if p.Stock <= p.LowStockThreshold {
				low = append(low, p)
			}
		}
		if len(low) == 0 {
			return "All products are adequately stocked. No items below threshold."
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%d product(s) need restocking:\n\n", len(low)))
		for _, p := range low {
			sb.WriteString(fmt.Sprintf("• [%s] %s\n", p.Id, p.Name))
			sb.WriteString(fmt.Sprintf("  Stock: %d %s | Threshold: %d | Price: $%.2f\n",
				p.Stock, unitName(p.Unit), p.LowStockThreshold, p.Price))
		}
		return sb.String()

	case "get_product":
		id, _ := args["id"].(string)
		for _, p := range products {
			if p.Id == id {
				return fmt.Sprintf("[%s] %s\nCategory: %s\nDescription: %s\nPrice: $%.2f\nStock: %d %s\nLow stock threshold: %d",
					p.Id, p.Name, categoryName(p.Category), p.Description,
					p.Price, p.Stock, unitName(p.Unit), p.LowStockThreshold)
			}
		}
		return fmt.Sprintf("Product %s not found.", id)

	case "get_inventory_summary":
		totalValue := 0.0
		categoryCounts := map[string]int{}
		lowStockCount := 0
		for _, p := range products {
			totalValue += p.Price * float64(p.Stock)
			categoryCounts[categoryName(p.Category)]++
			if p.Stock <= p.LowStockThreshold {
				lowStockCount++
			}
		}
		var sb strings.Builder
		sb.WriteString("ATX Technology Inventory Summary\n\n")
		sb.WriteString(fmt.Sprintf("Total products   : %d\n", len(products)))
		sb.WriteString(fmt.Sprintf("Total stock value: $%.2f\n", totalValue))
		sb.WriteString(fmt.Sprintf("Low stock alerts : %d\n\n", lowStockCount))
		sb.WriteString("Products by category:\n")
		for cat, count := range categoryCounts {
			sb.WriteString(fmt.Sprintf("  • %-12s %d\n", cat, count))
		}
		return sb.String()
	}

	return "Unknown tool."
}

// ─────────────────────────────────────────
//  Gemini API call with tool loop
// ─────────────────────────────────────────

func askGemini(apiKey, question string) (string, error) {
	contents := []GeminiContent{
		{Role: "user", Parts: []GeminiPart{{Text: question}}},
	}

	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=" + apiKey

	for {
		reqBody, _ := json.Marshal(GeminiRequest{
			Contents: contents,
			Tools:    geminiTools,
		})

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			return "", err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var geminiResp GeminiResponse
		if err := json.Unmarshal(body, &geminiResp); err != nil || len(geminiResp.Candidates) == 0 {
			return "", fmt.Errorf("bad response from Gemini: %s", string(body))
		}

		candidate := geminiResp.Candidates[0]
		responseParts := candidate.Content.Parts

		// Convert response parts to GeminiPart type
		var parts []GeminiPart
		for _, p := range responseParts {
			var fc *GeminiFunctionCall
			if p.FunctionCall != nil {
				fc = &GeminiFunctionCall{
					Name: p.FunctionCall.Name,
					Args: p.FunctionCall.Args,
				}
			}
			parts = append(parts, GeminiPart{
				Text:         p.Text,
				FunctionCall: fc,
			})
		}

		// Check if Gemini wants to call a tool
		hasFunctionCall := false
		for _, part := range parts {
			if part.FunctionCall != nil {
				hasFunctionCall = true
				break
			}
		}

		if hasFunctionCall {
			// Add Gemini's response to conversation
			contents = append(contents, GeminiContent{
				Role:  "model",
				Parts: parts,
			})

			// Execute each tool and collect results
			var resultParts []GeminiPart
			for _, part := range parts {
				if part.FunctionCall == nil {
					continue
				}
				fmt.Printf("  [MCP] Calling tool: %s\n", part.FunctionCall.Name)
				result := executeTool(part.FunctionCall.Name, part.FunctionCall.Args)
				resultParts = append(resultParts, GeminiPart{
					FunctionResp: &GeminiFunctionResp{
						Name:     part.FunctionCall.Name,
						Response: map[string]interface{}{"result": result},
					},
				})
			}

			// Send results back to Gemini
			contents = append(contents, GeminiContent{
				Role:  "user",
				Parts: resultParts,
			})
			continue
		}

		// Final text answer
		for _, part := range parts {
			if part.Text != "" {
				return part.Text, nil
			}
		}
		return "No response received.", nil
	}
}

// ─────────────────────────────────────────
//  Main — interactive chat loop
// ─────────────────────────────────────────

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: GEMINI_API_KEY environment variable not set.")
		fmt.Println("Run: $env:GEMINI_API_KEY=\"your-key-here\"")
		return
	}

	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║   ATX Technology — AI Inventory Agent    ║")
	fmt.Println("║   Powered by Gemini                      ║")
	fmt.Println("║   Type a question or 'exit' to quit      ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}
		question := strings.TrimSpace(scanner.Text())

		if question == "" {
			continue
		}
		if strings.ToLower(question) == "exit" {
			fmt.Println("Goodbye.")
			break
		}

		fmt.Println("  Thinking...")
		answer, err := askGemini(apiKey, question)
		if err != nil {
			fmt.Printf("  Error: %v\n\n", err)
			continue
		}

		fmt.Printf("\nAI: %s\n\n", answer)
		fmt.Println("──────────────────────────────────────────")
	}
}
