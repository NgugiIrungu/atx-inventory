package db

import (
	"fmt"
	"log"
	"time"
)

// Product — mirrors the MySQL products table
type Product struct {
	Id                string  `json:"id"`
	Name              string  `json:"name"`
	Description       string  `json:"description"`
	Category          int32   `json:"category"`
	Price             float64 `json:"price"`
	Stock             int32   `json:"stock"`
	Unit              int32   `json:"unit"`
	LowStockThreshold int32   `json:"low_stock_threshold"`
}

// counter — tracks the highest ATX-XXX number used
var counter int

// InitCounter — reads the highest existing ID from the database on startup
func InitCounter() {
	row := DB.QueryRow("SELECT COUNT(*) FROM products")
	row.Scan(&counter)
}

// GetAllProducts — returns every product in the database
func GetAllProducts() ([]*Product, error) {
	rows, err := DB.Query(`
		SELECT id, name, description, category, price, stock, unit, low_stock_threshold
		FROM products ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*Product
	for rows.Next() {
		p := &Product{}
		err := rows.Scan(&p.Id, &p.Name, &p.Description, &p.Category,
			&p.Price, &p.Stock, &p.Unit, &p.LowStockThreshold)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, nil
}

// GetProductByID — returns one product by its ID
func GetProductByID(id string) (*Product, error) {
	p := &Product{}
	err := DB.QueryRow(`
		SELECT id, name, description, category, price, stock, unit, low_stock_threshold
		FROM products WHERE id = ?
	`, id).Scan(&p.Id, &p.Name, &p.Description, &p.Category,
		&p.Price, &p.Stock, &p.Unit, &p.LowStockThreshold)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// AddProduct — inserts a new product and returns it with its assigned ID
func AddProduct(name, description string, category int32, price float64, stock, unit, threshold int32) (*Product, error) {
	counter++
	id := fmt.Sprintf("ATX-%03d", counter)

	_, err := DB.Exec(`
		INSERT INTO products (id, name, description, category, price, stock, unit, low_stock_threshold)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, id, name, description, category, price, stock, unit, threshold)
	if err != nil {
		// If ID collision, try incrementing
		counter++
		id = fmt.Sprintf("ATX-%03d", counter)
		_, err = DB.Exec(`
			INSERT INTO products (id, name, description, category, price, stock, unit, low_stock_threshold)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, id, name, description, category, price, stock, unit, threshold)
		if err != nil {
			return nil, err
		}
	}

	// Record initial stock in history
	AddHistory(id, stock, stock, 1, "Initial stock entry")

	log.Printf("✓ AddProduct: %s [%s]", name, id)
	return &Product{
		Id: id, Name: name, Description: description,
		Category: category, Price: price,
		Stock: stock, Unit: unit, LowStockThreshold: threshold,
	}, nil
}

// UpdateStock — adjusts stock by quantity and records the change
func UpdateStock(id string, quantity int32, reason int32, note string) (*Product, error) {
	// Update the stock
	_, err := DB.Exec(`
		UPDATE products SET stock = GREATEST(0, stock + ?) WHERE id = ?
	`, quantity, id)
	if err != nil {
		return nil, err
	}

	// Get the updated product
	p, err := GetProductByID(id)
	if err != nil {
		return nil, err
	}

	// Record the change
	AddHistory(id, quantity, p.Stock, reason, note)

	log.Printf("✓ UpdateStock: %s — change: %+d — new stock: %d", p.Name, quantity, p.Stock)
	return p, nil
}

// DeleteProduct — removes a product from the database
func DeleteProduct(id string) (string, error) {
	p, err := GetProductByID(id)
	if err != nil {
		return "", fmt.Errorf("product not found")
	}

	_, err = DB.Exec("DELETE FROM products WHERE id = ?", id)
	if err != nil {
		return "", err
	}

	log.Printf("✓ DeleteProduct: %s [%s]", p.Name, id)
	return p.Name, nil
}

// SeedProducts — fills the database with ATX Technology products if empty
func SeedProducts() {
	var count int
	DB.QueryRow("SELECT COUNT(*) FROM products").Scan(&count)
	if count > 0 {
		log.Printf("✓ Database already has %d products — skipping seed", count)
		return
	}

	log.Println("Seeding ATX Technology products...")

	type seed struct {
		name, desc string
		cat        int32
		price      float64
		stock      int32
		unit       int32
		threshold  int32
	}

	products := []seed{
		{"Single Mode Fiber Cable", "OS2 9/125 single mode fiber", 1, 1.20, 5000, 1, 500},
		{"OM3 Multimode Fiber Cable", "50/125 multimode, aqua", 1, 0.85, 3000, 1, 300},
		{"Fiber Patch Panel 24-Port", "LC duplex patch panel", 1, 45.00, 30, 2, 5},
		{"Cat6 LAN Cable", "U/UTP Cat6 solid copper", 2, 0.35, 8000, 1, 1000},
		{"Cat6A LAN Cable", "F/UTP Cat6A shielded", 2, 0.65, 4000, 1, 500},
		{"Cisco 24-Port Switch", "Catalyst 2960 Layer 2", 4, 320.00, 15, 2, 3},
		{"MikroTik Router", "hEX RB750Gr3", 3, 59.00, 25, 2, 5},
		{"LC/UPC Connector", "Single mode LC/UPC connector", 5, 0.80, 2000, 2, 200},
		{"RJ45 Connector", "Cat6 gold-plated RJ45", 5, 0.10, 5000, 3, 500},
		{"24-Port LAN Patch Panel", "Cat6 keystone patch panel", 2, 28.00, 20, 2, 5},
	}

	for _, p := range products {
		AddProduct(p.name, p.desc, p.cat, p.price, p.stock, p.unit, p.threshold)
	}

	log.Printf("✓ Seeded %d products", len(products))
	_ = time.Now()
}