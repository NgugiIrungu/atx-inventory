package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"
	pb "atx-inventory/pb"
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

	// GET /products — returns all products as JSON
	mux.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		s.mu.Lock()
		defer s.mu.Unlock()

		var list []*pb.Product
		for _, p := range s.products {
			list = append(list, p)
		}
		json.NewEncoder(w).Encode(list)
	})

	// GET /products/{id}/history — returns stock history for one product
	mux.HandleFunc("/history/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		id := r.URL.Path[len("/history/"):]
		s.mu.Lock()
		defer s.mu.Unlock()

		history := s.history[id]
		json.NewEncoder(w).Encode(history)
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
}