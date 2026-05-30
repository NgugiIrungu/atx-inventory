package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	pb "atx-inventory/pb"
)

const dataFile = "data/inventory.json"

// ─────────────────────────────────────────
//  JSONProduct — what we save to disk
//  (we can't save protobuf structs directly)
// ─────────────────────────────────────────

type JSONProduct struct {
	Id                string  `json:"id"`
	Name              string  `json:"name"`
	Description       string  `json:"description"`
	Category          int32   `json:"category"`
	Price             float64 `json:"price"`
	Stock             int32   `json:"stock"`
	Unit              int32   `json:"unit"`
	LowStockThreshold int32   `json:"low_stock_threshold"`
}

type JSONHistory struct {
	ProductId  string `json:"product_id"`
	Quantity   int32  `json:"quantity"`
	StockAfter int32  `json:"stock_after"`
	Reason     int32  `json:"reason"`
	Note       string `json:"note"`
	Timestamp  string `json:"timestamp"`
}

type JSONStore struct {
	Products map[string]*JSONProduct   `json:"products"`
	History  map[string][]*JSONHistory `json:"history"`
	Counter  int                       `json:"counter"`
}

// ─────────────────────────────────────────
//  store — in-memory + file persistence
// ─────────────────────────────────────────

type store struct {
	mu       sync.Mutex
	products map[string]*pb.Product
	history  map[string][]*pb.StockHistory
	counter  int
}

// newStore — loads from file if it exists, seeds fresh data if not
func newStore() *store {
	s := &store{
		products: make(map[string]*pb.Product),
		history:  make(map[string][]*pb.StockHistory),
	}

	if _, err := os.Stat(dataFile); err == nil {
		// File exists — load from disk
		if err := s.load(); err != nil {
			log.Printf("Warning: could not load data file: %v — starting fresh", err)
			s.seed()
		} else {
			log.Printf("✓ Loaded inventory from %s (%d products)", dataFile, len(s.products))
		}
	} else {
		// No file yet — seed with ATX products and save
		log.Printf("No data file found — seeding fresh inventory")
		s.seed()
		s.save()
	}

	return s
}

// seed — populates the store with real ATX Technology products
func (s *store) seed() {
	products := []struct {
		name, desc string
		cat        pb.Category
		price      float64
		stock      int32
		unit       pb.Unit
		threshold  int32
	}{
		{"Single Mode Fiber Cable", "OS2 9/125 single mode fiber", pb.Category_FIBER, 1.20, 5000, pb.Unit_METRES, 500},
		{"OM3 Multimode Fiber Cable", "50/125 multimode, aqua", pb.Category_FIBER, 0.85, 3000, pb.Unit_METRES, 300},
		{"Fiber Patch Panel 24-Port", "LC duplex patch panel", pb.Category_FIBER, 45.00, 30, pb.Unit_PIECES, 5},
		{"Cat6 LAN Cable", "U/UTP Cat6 solid copper", pb.Category_LAN, 0.35, 8000, pb.Unit_METRES, 1000},
		{"Cat6A LAN Cable", "F/UTP Cat6A shielded", pb.Category_LAN, 0.65, 4000, pb.Unit_METRES, 500},
		{"Cisco 24-Port Switch", "Catalyst 2960 Layer 2", pb.Category_SWITCHES, 320.00, 15, pb.Unit_PIECES, 3},
		{"MikroTik Router", "hEX RB750Gr3", pb.Category_ROUTERS, 59.00, 25, pb.Unit_PIECES, 5},
		{"LC/UPC Connector", "Single mode LC/UPC connector", pb.Category_CONNECTORS, 0.80, 2000, pb.Unit_PIECES, 200},
		{"RJ45 Connector", "Cat6 gold-plated RJ45", pb.Category_CONNECTORS, 0.10, 5000, pb.Unit_BOXES, 500},
		{"24-Port LAN Patch Panel", "Cat6 keystone patch panel", pb.Category_LAN, 28.00, 20, pb.Unit_PIECES, 5},
	}

	for _, p := range products {
		s.counter++
		id := fmt.Sprintf("ATX-%03d", s.counter)
		s.products[id] = &pb.Product{
			Id: id, Name: p.name, Description: p.desc,
			Category: p.cat, Price: p.price,
			Stock: p.stock, Unit: p.unit,
			LowStockThreshold: p.threshold,
		}
	}
}

// save — writes current state to inventory.json
func (s *store) save() {
	js := &JSONStore{
		Products: make(map[string]*JSONProduct),
		History:  make(map[string][]*JSONHistory),
		Counter:  s.counter,
	}

	for id, p := range s.products {
		js.Products[id] = &JSONProduct{
			Id: p.Id, Name: p.Name, Description: p.Description,
			Category: int32(p.Category), Price: p.Price,
			Stock: p.Stock, Unit: int32(p.Unit),
			LowStockThreshold: p.LowStockThreshold,
		}
	}

	for id, entries := range s.history {
		for _, h := range entries {
			js.History[id] = append(js.History[id], &JSONHistory{
				ProductId: h.ProductId, Quantity: h.Quantity,
				StockAfter: h.StockAfter, Reason: int32(h.Reason),
				Note: h.Note, Timestamp: h.Timestamp,
			})
		}
	}

	data, err := json.MarshalIndent(js, "", "  ")
	if err != nil {
		log.Printf("Error marshalling data: %v", err)
		return
	}

	if err := os.WriteFile(dataFile, data, 0644); err != nil {
		log.Printf("Error writing data file: %v", err)
		return
	}

	log.Printf("✓ Saved to %s", dataFile)
}

// load — reads inventory.json and populates the store
func (s *store) load() error {
	data, err := os.ReadFile(dataFile)
	if err != nil {
		return err
	}

	var js JSONStore
	if err := json.Unmarshal(data, &js); err != nil {
		return err
	}

	s.counter = js.Counter

	for id, jp := range js.Products {
		s.products[id] = &pb.Product{
			Id: jp.Id, Name: jp.Name, Description: jp.Description,
			Category: pb.Category(jp.Category), Price: jp.Price,
			Stock: jp.Stock, Unit: pb.Unit(jp.Unit),
			LowStockThreshold: jp.LowStockThreshold,
		}
	}

	for id, entries := range js.History {
		for _, jh := range entries {
			s.history[id] = append(s.history[id], &pb.StockHistory{
				ProductId: jh.ProductId, Quantity: jh.Quantity,
				StockAfter: jh.StockAfter, Reason: pb.ChangeReason(jh.Reason),
				Note: jh.Note, Timestamp: jh.Timestamp,
			})
		}
	}

	return nil
}
