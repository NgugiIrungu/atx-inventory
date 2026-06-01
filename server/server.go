package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
	"bytes"
	"os"
	"strings"
	

	pb "atx-inventory/pb"

	"google.golang.org/grpc"
)

// ─────────────────────────────────────────
//  inventoryServer — implements gRPC service
// ─────────────────────────────────────────

type inventoryServer struct {
	pb.UnimplementedInventoryServiceServer
	store *store
}

func (s *inventoryServer) AddProduct(_ context.Context, req *pb.AddProductRequest) (*pb.ProductResponse, error) {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	s.store.counter++
	id := fmt.Sprintf("ATX-%03d", s.store.counter)

	product := &pb.Product{
		Id: id, Name: req.Name, Description: req.Description,
		Category: req.Category, Price: req.Price,
		Stock: req.Stock, Unit: req.Unit,
		LowStockThreshold: req.LowStockThreshold,
	}

	s.store.products[id] = product
	recordHistory(s.store, id, req.Stock, req.Stock, pb.ChangeReason_RESTOCK, "Initial stock entry")
	s.store.save()

	log.Printf("✓ AddProduct: %s [%s]", product.Name, id)
	return &pb.ProductResponse{Success: true, Message: "Product added successfully", Product: product}, nil
}

func (s *inventoryServer) GetProduct(_ context.Context, req *pb.GetProductRequest) (*pb.ProductResponse, error) {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	p, ok := s.store.products[req.Id]
	if !ok {
		return &pb.ProductResponse{Success: false, Message: "Product not found"}, nil
	}
	return &pb.ProductResponse{Success: true, Product: p}, nil
}

func (s *inventoryServer) UpdateStock(_ context.Context, req *pb.UpdateStockRequest) (*pb.ProductResponse, error) {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	p, ok := s.store.products[req.Id]
	if !ok {
		return &pb.ProductResponse{Success: false, Message: "Product not found"}, nil
	}

	p.Stock += req.Quantity
	if p.Stock < 0 {
		p.Stock = 0
	}

	recordHistory(s.store, req.Id, req.Quantity, p.Stock, req.Reason, req.Note)
	s.store.save()

	log.Printf("✓ UpdateStock: %s — change: %+d — new stock: %d", p.Name, req.Quantity, p.Stock)
	return &pb.ProductResponse{Success: true, Message: "Stock updated", Product: p}, nil
}

func (s *inventoryServer) DeleteProduct(_ context.Context, req *pb.DeleteProductRequest) (*pb.DeleteResponse, error) {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	p, ok := s.store.products[req.Id]
	if !ok {
		return &pb.DeleteResponse{Success: false, Message: "Product not found"}, nil
	}
	delete(s.store.products, req.Id)
	s.store.save()

	log.Printf("✓ DeleteProduct: %s [%s]", p.Name, req.Id)
	return &pb.DeleteResponse{Success: true, Message: fmt.Sprintf("%s deleted successfully", p.Name)}, nil
}

func (s *inventoryServer) ListProducts(_ *pb.ListProductsRequest, stream pb.InventoryService_ListProductsServer) error {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	for _, p := range s.store.products {
		if err := stream.Send(&pb.ProductResponse{Success: true, Product: p}); err != nil {
			return err
		}
	}
	return nil
}

func (s *inventoryServer) GetStockHistory(_ context.Context, req *pb.GetStockHistoryRequest) (*pb.StockHistoryResponse, error) {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	p, ok := s.store.products[req.Id]
	if !ok {
		return nil, fmt.Errorf("product not found: %s", req.Id)
	}
	return &pb.StockHistoryResponse{
		ProductId: req.Id,
		Name:      p.Name,
		History:   s.store.history[req.Id],
	}, nil
}

func recordHistory(s *store, productId string, qty, stockAfter int32, reason pb.ChangeReason, note string) {
	s.history[productId] = append(s.history[productId], &pb.StockHistory{
		ProductId:  productId,
		Quantity:   qty,
		StockAfter: stockAfter,
		Reason:     reason,
		Note:       note,
		Timestamp:  time.Now().Format("2006-01-02 15:04:05"),
	})
}

// ─────────────────────────────────────────
//  HTTP layer — so Vue.js can talk to us
// ─────────────────────────────────────────

func startHTTPServer(s *store) {
	mux := http.NewServeMux()

	// Shared CORS headers for all responses
	cors := func(w http.ResponseWriter) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	}

	// GET /products — list all products
	// POST /products — add a new product
	mux.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "OPTIONS" {
			return
		}

		if r.Method == "GET" {
			s.mu.Lock()
			defer s.mu.Unlock()
			var list []*pb.Product
			for _, p := range s.products {
				list = append(list, p)
			}
			json.NewEncoder(w).Encode(list)
			return
		}

		if r.Method == "POST" {
			var req pb.AddProductRequest
			json.NewDecoder(r.Body).Decode(&req)

			s.mu.Lock()
			defer s.mu.Unlock()

			s.counter++
			id := fmt.Sprintf("ATX-%03d", s.counter)
			product := &pb.Product{
				Id: id, Name: req.Name, Description: req.Description,
				Category: req.Category, Price: req.Price,
				Stock: req.Stock, Unit: req.Unit,
				LowStockThreshold: req.LowStockThreshold,
			}
			s.products[id] = product
			recordHistory(s, id, req.Stock, req.Stock, pb.ChangeReason_RESTOCK, "Added via dashboard")
			s.save()

			log.Printf("✓ HTTP AddProduct: %s [%s]", product.Name, id)
			json.NewEncoder(w).Encode(product)
		}
	})

	// DELETE /products/{id} — delete a product
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

		s.mu.Lock()
		defer s.mu.Unlock()

		p, ok := s.products[id]
		if !ok {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Product not found"})
			return
		}
		delete(s.products, id)
		s.save()
		log.Printf("✓ HTTP DeleteProduct: %s [%s]", p.Name, id)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": p.Name + " deleted"})
	})

	// POST /stock — update stock for a product
	mux.HandleFunc("/stock", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "OPTIONS" {
			return
		}

		var req struct {
			Id       string `json:"id"`
			Quantity int32  `json:"quantity"`
			Reason   int32  `json:"reason"`
			Note     string `json:"note"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		s.mu.Lock()
		defer s.mu.Unlock()

		p, ok := s.products[req.Id]
		if !ok {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Product not found"})
			return
		}

		p.Stock += req.Quantity
		if p.Stock < 0 {
			p.Stock = 0
		}

		recordHistory(s, req.Id, req.Quantity, p.Stock, pb.ChangeReason(req.Reason), req.Note)
		s.save()

		log.Printf("✓ HTTP UpdateStock: %s — change: %+d — new stock: %d", p.Name, req.Quantity, p.Stock)
		json.NewEncoder(w).Encode(p)
	})

	// GET /history/{id} — stock history for a product
	mux.HandleFunc("/history/", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		w.Header().Set("Content-Type", "application/json")

		id := r.URL.Path[len("/history/"):]
		s.mu.Lock()
		defer s.mu.Unlock()
		json.NewEncoder(w).Encode(s.history[id])
	})
	// POST /ai — receives a question, calls the AI agent, returns the answer
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

    // Build product context from current inventory
    s.mu.Lock()
    var sb strings.Builder
    sb.WriteString("ATX Technology current inventory:\n")
    for _, p := range s.products {
        status := "OK"
        if p.Stock <= p.LowStockThreshold {
            status = "LOW STOCK"
        }
        sb.WriteString(fmt.Sprintf("- [%s] %s | Category: %d | Price: $%.2f | Stock: %d | Status: %s\n",
            p.Id, p.Name, p.Category, p.Price, p.Stock, status))
    }
    s.mu.Unlock()

    prompt := fmt.Sprintf(`You are an inventory assistant for ATX Technology, a networking equipment company.
Here is the current inventory:

%s

Answer this question clearly and concisely: %s`, sb.String(), req.Question)

    answer := callAI(prompt)
    json.NewEncoder(w).Encode(map[string]string{
        "answer": answer,
    })
})

	log.Printf("✓ HTTP server listening on :8080")
	http.ListenAndServe(":8080", mux)
}

// ─────────────────────────────────────────
//  Main — start both gRPC and HTTP servers
// ─────────────────────────────────────────

func main() {
	st := newStore()

	// Start HTTP server in background
	go startHTTPServer(st)

	// Start gRPC server in foreground
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal("Failed to listen:", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterInventoryServiceServer(grpcServer, &inventoryServer{store: st})

	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║   ATX Technology Inventory Server    ║")
	fmt.Println("║   gRPC  → :50051                     ║")
	fmt.Println("║   HTTP  → :8080                      ║")
	fmt.Println("╚══════════════════════════════════════╝")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal("Failed to serve:", err)
	}
	func callAI(prompt string) string {
    apiKey := os.Getenv("ANTHROPIC_API_KEY")
    geminiKey := os.Getenv("GEMINI_API_KEY")

    if geminiKey != "" {
        return callGemini(geminiKey, prompt)
    }
    if apiKey != "" {
        return callClaude(apiKey, prompt)
    }
    return "No AI API key set. Set ANTHROPIC_API_KEY or GEMINI_API_KEY."
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
    try := result["candidates"].([]interface{})[0].(map[string]interface{})["content"].(map[string]interface{})["parts"].([]interface{})[0].(map[string]interface{})["text"].(string)
    return try
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
}
