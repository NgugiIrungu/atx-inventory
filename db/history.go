package db

import "log"

// StockHistory — mirrors the stock_history table
type StockHistory struct {
	Id         int    `json:"id"`
	ProductId  string `json:"product_id"`
	Quantity   int32  `json:"quantity"`
	StockAfter int32  `json:"stock_after"`
	Reason     int32  `json:"reason"`
	Note       string `json:"note"`
	Timestamp  string `json:"timestamp"`
}

// AddHistory — inserts a new stock history record
func AddHistory(productId string, quantity, stockAfter, reason int32, note string) {
	_, err := DB.Exec(`
		INSERT INTO stock_history (product_id, quantity, stock_after, reason, note)
		VALUES (?, ?, ?, ?, ?)
	`, productId, quantity, stockAfter, reason, note)
	if err != nil {
		log.Printf("Warning: failed to record history: %v", err)
	}
}

// GetHistory — returns all stock changes for a product
func GetHistory(productId string) ([]*StockHistory, error) {
	rows, err := DB.Query(`
		SELECT id, product_id, quantity, stock_after, reason, note,
		DATE_FORMAT(timestamp, '%Y-%m-%d %H:%i:%s') as timestamp
		FROM stock_history
		WHERE product_id = ?
		ORDER BY timestamp ASC
	`, productId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []*StockHistory
	for rows.Next() {
		h := &StockHistory{}
		err := rows.Scan(&h.Id, &h.ProductId, &h.Quantity, &h.StockAfter,
			&h.Reason, &h.Note, &h.Timestamp)
		if err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, nil
}