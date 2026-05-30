package main

import (
	"context"
	"fmt"
	"io"
	"log"

	pb "atx-inventory/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connect to the server
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer conn.Close()

	client := pb.NewInventoryServiceClient(conn)
	ctx := context.Background()

	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║   ATX Technology — Inventory Client  ║")
	fmt.Println("╚══════════════════════════════════════╝")

	// ── 1. LIST all products ──────────────────────────────
	fmt.Println("\n📦 CURRENT INVENTORY")
	fmt.Println("─────────────────────────────────────────────────────────────────")
	fmt.Printf("%-10s %-30s %-12s %8s %8s %s\n", "ID", "Name", "Category", "Price", "Stock", "Status")
	fmt.Println("─────────────────────────────────────────────────────────────────")

	stream, err := client.ListProducts(ctx, &pb.ListProductsRequest{})
	if err != nil {
		log.Fatal("ListProducts failed:", err)
	}
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("Stream error:", err)
		}
		p := res.Product
		status := "✓ OK"
		if p.Stock <= p.LowStockThreshold {
			status = "⚠ LOW"
		}
		fmt.Printf("%-10s %-30s %-12s %8.2f %8d %s\n",
			p.Id, p.Name, categoryName(p.Category), p.Price, p.Stock, status)
	}

	// ── 2. ADD a new product ──────────────────────────────
	fmt.Println("\n➕ ADDING NEW PRODUCT")
	addRes, err := client.AddProduct(ctx, &pb.AddProductRequest{
		Name:              "Fiber Optic Splice Closure",
		Description:       "Dome type, 48 fibers, IP68",
		Category:          pb.Category_FIBER,
		Price:             18.50,
		Stock:             40,
		Unit:              pb.Unit_PIECES,
		LowStockThreshold: 5,
	})
	if err != nil {
		log.Fatal("AddProduct failed:", err)
	}
	fmt.Printf("  %s — ID assigned: %s\n", addRes.Message, addRes.Product.Id)
	newID := addRes.Product.Id

	// ── 3. UPDATE stock — restock ─────────────────────────
	fmt.Println("\n🔄 RESTOCKING Cat6 LAN Cable (+2000 metres)")
	updateRes, err := client.UpdateStock(ctx, &pb.UpdateStockRequest{
		Id:       "ATX-004",
		Quantity: 2000,
		Reason:   pb.ChangeReason_RESTOCK,
		Note:     "New shipment received from supplier",
	})
	if err != nil {
		log.Fatal("UpdateStock failed:", err)
	}
	fmt.Printf("  %s — new stock: %d metres\n", updateRes.Message, updateRes.Product.Stock)

	// ── 4. UPDATE stock — sale ────────────────────────────
	fmt.Println("\n💸 RECORDING SALE — MikroTik Router (-3 units)")
	saleRes, err := client.UpdateStock(ctx, &pb.UpdateStockRequest{
		Id:       "ATX-007",
		Quantity: -3,
		Reason:   pb.ChangeReason_SALE,
		Note:     "Sold to Kampala client - invoice INV-2024-089",
	})
	if err != nil {
		log.Fatal("UpdateStock failed:", err)
	}
	fmt.Printf("  %s — remaining stock: %d\n", saleRes.Message, saleRes.Product.Stock)

	// ── 5. VIEW stock history ─────────────────────────────
	fmt.Println("\n📋 STOCK HISTORY — Cat6 LAN Cable (ATX-004)")
	histRes, err := client.GetStockHistory(ctx, &pb.GetStockHistoryRequest{Id: "ATX-004"})
	if err != nil {
		log.Fatal("GetStockHistory failed:", err)
	}
	fmt.Printf("  Product: %s\n", histRes.Name)
	for i, h := range histRes.History {
		change := fmt.Sprintf("%+d", h.Quantity)
		fmt.Printf("  [%d] %s | Change: %-6s | Stock after: %-6d | %s | %s\n",
			i+1, h.Timestamp, change, h.StockAfter, reasonName(h.Reason), h.Note)
	}

	// ── 6. DELETE the product we just added ───────────────
	fmt.Printf("\n🗑  DELETING new product [%s]\n", newID)
	delRes, err := client.DeleteProduct(ctx, &pb.DeleteProductRequest{Id: newID})
	if err != nil {
		log.Fatal("DeleteProduct failed:", err)
	}
	fmt.Printf("  %s\n", delRes.Message)

	fmt.Println("\n══════════════════════════════════════════")
	fmt.Println("  All operations completed successfully")
	fmt.Println("══════════════════════════════════════════")
}

// Helper to print category names cleanly
func categoryName(c pb.Category) string {
	switch c {
	case pb.Category_FIBER:
		return "Fiber"
	case pb.Category_LAN:
		return "LAN"
	case pb.Category_ROUTERS:
		return "Routers"
	case pb.Category_SWITCHES:
		return "Switches"
	case pb.Category_CONNECTORS:
		return "Connectors"
	default:
		return "Unknown"
	}
}

// Helper to print change reason names cleanly
func reasonName(r pb.ChangeReason) string {
	switch r {
	case pb.ChangeReason_RESTOCK:
		return "RESTOCK"
	case pb.ChangeReason_SALE:
		return "SALE"
	case pb.ChangeReason_DAMAGE:
		return "DAMAGE"
	case pb.ChangeReason_MANUAL_ADJUSTMENT:
		return "ADJUSTMENT"
	default:
		return "UNKNOWN"
	}
}
