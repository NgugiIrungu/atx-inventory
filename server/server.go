package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"google.golang.org/grpc"
	db "atx-inventory/db"
	pb "atx-inventory/pb"
)

// ─────────────────────────────────────────
//  gRPC server
// ─────────────────────────────────────────

type inventoryServer struct {
	pb.UnimplementedInventoryServiceServer
}

func (s *inventoryServer) AddProduct(_ context.Context, req *pb.AddProductRequest) (*pb.ProductResponse, error) {
	p, err := db.AddProduct(
		req.Name, req.Description,
		int32(req.Category), req.Price,
		req.Stock, int32(req.Unit), req.LowStockThreshold,
	)
	if err != nil {
		return &pb.ProductResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.ProductResponse{
		Success: true,
		Message: "Product added successfully",
		Product: toProto(p),
	}, nil
}

func (s *inventoryServer) GetProduct(_ context.Context, req *pb.GetProductRequest) (*pb.ProductResponse, error) {
	p, err := db.GetProductByID(req.Id)
	if err != nil {
		return &pb.ProductResponse{Success: false, Message: "Product not found"}, nil
	}
	return &pb.ProductResponse{Success: true, Product: toProto(p)}, nil
}

func (s *inventoryServer) UpdateStock(_ context.Context, req *pb.UpdateStockRequest) (*pb.ProductResponse, error) {
	p, err := db.UpdateStock(req.Id, req.Quantity, int32(req.Reason), req.Note)
	if err != nil {
		return &pb.ProductResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.ProductResponse{Success: true, Message: "Stock updated", Product: toProto(p)}, nil
}

func (s *inventoryServer) DeleteProduct(_ context.Context, req *pb.DeleteProductRequest) (*pb.DeleteResponse, error) {
	name, err := db.DeleteProduct(req.Id)
	if err != nil {
		return &pb.DeleteResponse{Success: false, Message: err.Error()}, nil
	}
	return &pb.DeleteResponse{Success: true, Message: name + " deleted successfully"}, nil
}

func (s *inventoryServer) ListProducts(_ *pb.ListProductsRequest, stream pb.InventoryService_ListProductsServer) error {
	products, err := db.GetAllProducts()
	if err != nil {
		return err
	}
	for _, p := range products {
		if err := stream.Send(&pb.ProductResponse{Success: true, Product: toProto(p)}); err != nil {
			return err
		}
	}
	return nil
}

func (s *inventoryServer) GetStockHistory(_ context.Context, req *pb.GetStockHistoryRequest) (*pb.StockHistoryResponse, error) {
	p, err := db.GetProductByID(req.Id)
	if err != nil {
		return nil, fmt.Errorf("product not found: %s", req.Id)
	}
	history, err := db.GetHistory(req.Id)
	if err != nil {
		return nil, err
	}
	var pbHistory []*pb.StockHistory
	for _, h := range history {
		pbHistory = append(pbHistory, &pb.StockHistory{
			ProductId:  h.ProductId,
			Quantity:   h.Quantity,
			StockAfter: h.StockAfter,
			Reason:     pb.ChangeReason(h.Reason),
			Note:       h.Note,
			Timestamp:  h.Timestamp,
		})
	}
	return &pb.StockHistoryResponse{
		ProductId: req.Id,
		Name:      p.Name,
		History:   pbHistory,
	}, nil
}

// toProto — converts a db.Product to a pb.Product
func toProto(p *db.Product) *pb.Product {
	return &pb.Product{
		Id:                p.Id,
		Name:              p.Name,
		Description:       p.Description,
		Category:          pb.Category(p.Category),
		Price:             p.Price,
		Stock:             p.Stock,
		Unit:              pb.Unit(p.Unit),
		LowStockThreshold: p.LowStockThreshold,
	}
}

// ─────────────────────────────────────────
//  JWT helpers
// ─────────────────────────────────────────

var jwtSecret = []byte("atx-secret-key-2024")

func generateToken(userID int, username, role string) string {
	// Simple token: base64(userID|username|role|timestamp)
	// In production use a proper JWT library
	payload := fmt.Sprintf("%d|%s|%s|%d", userID, username, role, timeNow())
	return encodeBase64(payload)
}

func parseToken(token string) (int, string, string, bool) {
	payload := decodeBase64(token)
	parts := strings.Split(payload, "|")
	if len(parts) != 4 {
		return 0, "", "", false
	}
	var id int
	fmt.Sscanf(parts[0], "%d", &id)
	return id, parts[1], parts[2], true
}

func timeNow() int64 {
	return 1000000
}

func encodeBase64(s string) string {
	encoded := make([]byte, len(s)*2)
	for i, c := range s {
		encoded[i*2] = byte(c) + 1
		encoded[i*2+1] = byte(c>>8) + 1
	}
	return fmt.Sprintf("%x", encoded[:len(s)*2])
}

func decodeBase64(s string) string {
	if len(s)%2 != 0 {
		return ""
	}
	result := make([]byte, len(s)/2)
	for i := range result {
		if i*4+3 >= len(s) {
			break
		}
		var b byte
		fmt.Sscanf(s[i*4:i*4+2], "%02x", &b)
		result[i] = b - 1
	}
	return string(result)
}

// ─────────────────────────────────────────
//  HTTP server
// ─────────────────────────────────────────

func startHTTPServer() {
	mux := http.NewServeMux()

	cors := func(w http.ResponseWriter) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	}

	// ── Auth endpoints ──────────────────────────────────

	// POST /register — create a new user
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "OPTIONS" {
			return
		}
		var req struct {
			Username string `json:"username"`
			Email    string `json:"email"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Role == "" {
			req.Role = "user"
		}
		user, err := db.CreateUser(req.Username, req.Email, req.Password, req.Role)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false, "message": "Username or email already exists",
			})
			return
		}
		token := generateToken(user.Id, user.Username, user.Role)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"token":   token,
			"user":    user,
		})
	})

	// POST /login — authenticate a user
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "OPTIONS" {
			return
		}
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		user, err := db.AuthenticateUser(req.Username, req.Password)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false, "message": "Invalid username or password",
			})
			return
		}
		token := generateToken(user.Id, user.Username, user.Role)
		log.Printf("✓ Login: %s [%s]", user.Username, user.Role)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"token":   token,
			"user":    user,
		})
	})

	// ── Product endpoints ───────────────────────────────

	// GET /products — list all
	// POST /products — add new
	mux.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "OPTIONS" {
			return
		}

		if r.Method == "GET" {
			products, err := db.GetAllProducts()
			if err != nil {
				json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
				return
			}
			json.NewEncoder(w).Encode(products)
			return
		}

		if r.Method == "POST" {
			// Check auth
			_, _, role, ok := getTokenFromRequest(r)
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]interface{}{"message": "Unauthorized"})
				return
			}
			if role != "admin" {
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{"message": "Admin access required"})
				return
			}

			var req struct {
				Name              string  `json:"name"`
				Description       string  `json:"description"`
				Category          int32   `json:"category"`
				Price             float64 `json:"price"`
				Stock             int32   `json:"stock"`
				Unit              int32   `json:"unit"`
				LowStockThreshold int32   `json:"low_stock_threshold"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			p, err := db.AddProduct(req.Name, req.Description, req.Category,
				req.Price, req.Stock, req.Unit, req.LowStockThreshold)
			if err != nil {
				json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
				return
			}
			json.NewEncoder(w).Encode(p)
		}
	})

	// DELETE /products/{id}
	mux.HandleFunc("/products/", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "OPTIONS" {
			return
		}

		id := r.URL.Path[len("/products/"):]
		if id == "" {
			return
		}

		_, _, role, ok := getTokenFromRequest(r)
		if !ok || role != "admin" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{"message": "Admin access required"})
			return
		}

		name, err := db.DeleteProduct(id)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": name + " deleted"})
	})

	// POST /stock — update stock
	mux.HandleFunc("/stock", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "OPTIONS" {
			return
		}

		_, _, _, ok := getTokenFromRequest(r)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{"message": "Unauthorized"})
			return
		}

		var req struct {
			Id       string `json:"id"`
			Quantity int32  `json:"quantity"`
			Reason   int32  `json:"reason"`
			Note     string `json:"note"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		p, err := db.UpdateStock(req.Id, req.Quantity, req.Reason, req.Note)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(p)
	})

	// GET /history/{id}
	mux.HandleFunc("/history/", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		w.Header().Set("Content-Type", "application/json")
		id := r.URL.Path[len("/history/"):]
		history, err := db.GetHistory(id)
		if err != nil {
			json.NewEncoder(w).Encode([]interface{}{})
			return
		}
		json.NewEncoder(w).Encode(history)
	})

	// POST /ai — AI question endpoint
	mux.HandleFunc("/ai", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "OPTIONS" {
			return
		}

		var req struct {
			Question string `json:"question"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		products, _ := db.GetAllProducts()
		var sb strings.Builder
		sb.WriteString("ATX Technology current inventory:\n")
		for _, p := range products {
			status := "OK"
			if p.Stock <= p.LowStockThreshold {
				status = "LOW STOCK"
			}
			sb.WriteString(fmt.Sprintf("- [%s] %s | Category: %d | Price: $%.2f | Stock: %d | Status: %s\n",
				p.Id, p.Name, p.Category, p.Price, p.Stock, status))
		}

		prompt := fmt.Sprintf(`You are an inventory assistant for ATX Technology, a networking equipment company.
Here is the current inventory:

%s

Answer this question clearly and concisely: %s`, sb.String(), req.Question)

		answer := callAI(prompt)
		json.NewEncoder(w).Encode(map[string]string{"answer": answer})
	})

	log.Printf("✓ HTTP server listening on :8080")
	http.ListenAndServe(":8080", mux)
}

// getTokenFromRequest — extracts and parses the Authorization header
func getTokenFromRequest(r *http.Request) (int, string, string, bool) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return 0, "", "", false
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	return parseToken(token)
}

// ─────────────────────────────────────────
//  AI helpers
// ─────────────────────────────────────────

func callAI(prompt string) string {
	geminiKey := os.Getenv("GEMINI_API_KEY")
	claudeKey := os.Getenv("ANTHROPIC_API_KEY")

	if geminiKey != "" {
		return callGemini(geminiKey, prompt)
	}
	if claudeKey != "" {
		return callClaude(claudeKey, prompt)
	}
	return "No AI API key set. Set GEMINI_API_KEY or ANTHROPIC_API_KEY."
}

func callGemini(apiKey, prompt string) string {
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=" + apiKey
	body, _ := json.Marshal(map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{{"text": prompt}}},
		},
	})
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "Error calling Gemini: " + err.Error()
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	candidates, ok := result["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return "No response from Gemini."
	}
	content := candidates[0].(map[string]interface{})["content"].(map[string]interface{})
	parts := content["parts"].([]interface{})
	return parts[0].(map[string]interface{})["text"].(string)
}

func callClaude(apiKey, prompt string) string {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":      "claude-sonnet-4-20250514",
		"max_tokens": 500,
		"messages":   []map[string]string{{"role": "user", "content": prompt}},
	})
	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "Error calling Claude: " + err.Error()
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	content := result["content"].([]interface{})[0].(map[string]interface{})["text"].(string)
	return content
}

// ─────────────────────────────────────────
//  Main
// ─────────────────────────────────────────

func main() {
	// Connect to MySQL
	err := db.Connect("root", "ngugi", "localhost:3306", "atx_inventory")
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	// Seed products if database is empty
	db.SeedProducts()
	db.InitCounter()
	db.UpdateAdminPassword()

	// Start HTTP server in background
	go startHTTPServer()

	// Start gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal("Failed to listen:", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterInventoryServiceServer(grpcServer, &inventoryServer{})

	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║   ATX Technology Inventory Server    ║")
	fmt.Println("║   gRPC  → :50051                     ║")
	fmt.Println("║   HTTP  → :8080                      ║")
	fmt.Println("║   DB    → MySQL atx_inventory        ║")
	fmt.Println("╚══════════════════════════════════════╝")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal("Failed to serve:", err)
	}
}