package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Models remain the same as before, but let's add trade_id to BrokerTrade
type BrokerTrade struct {
	TradeID         string
	DealDate        time.Time
	BrokerCode      string
	ISIN            string
	BuySellFlag     string
	Quantity        int
	Price           float64
	Cost            float64  // renamed from NetAmount for clarity
	BrokerageAmount float64
	SettlementDate  time.Time
	STT             float64
	ExchangeCode    string
	DepositoryCode  string
}

func initDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	// Create tables with necessary indexes
	createTables := `
	CREATE TABLE IF NOT EXISTS client_orders (
		order_id TEXT PRIMARY KEY,
		client_id TEXT,
		isin TEXT,
		quantity INTEGER,
		order_price REAL,
		order_date DATE,
		UNIQUE(isin, order_date, quantity)
	);
	CREATE INDEX IF NOT EXISTS idx_orders_isin_date ON client_orders(isin, order_date);

	CREATE TABLE IF NOT EXISTS broker_trades (
		trade_id TEXT PRIMARY KEY,
		deal_date DATE,
		broker_code TEXT,
		isin TEXT,
		buy_sell_flag TEXT,
		quantity INTEGER,
		price REAL,
		cost REAL,
		brokerage_amount REAL,
		settlement_date DATE,
		stt REAL,
		exchange_code TEXT,
		depository_code TEXT,
		UNIQUE(isin, deal_date, quantity)
	);
	CREATE INDEX IF NOT EXISTS idx_trades_isin_date ON broker_trades(isin, deal_date);

	CREATE TABLE IF NOT EXISTS reconciliation (
		order_id TEXT,
		client_id TEXT,
		trade_id TEXT,
		symbol TEXT,
		trade_date DATE,
		matched_quantity INTEGER,
		unmatched_quantity INTEGER,
		status TEXT,
		avg_price REAL,
		total_cost REAL,
		execution_slippage REAL,
		FOREIGN KEY(order_id) REFERENCES client_orders(order_id),
		FOREIGN KEY(trade_id) REFERENCES broker_trades(trade_id)
	);`

	_, err = db.Exec(createTables)
	return db, err
}

func performReconciliation(db *sql.DB) error {
	reconciliationQuery := `
	WITH broker_ranked AS (
		SELECT *,
			ROW_NUMBER() OVER (PARTITION BY isin, deal_date ORDER BY quantity DESC) AS broker_rank
		FROM broker_trades
	),
	client_ranked AS (
		SELECT *,
			ROW_NUMBER() OVER (PARTITION BY isin, order_date ORDER BY quantity DESC) AS client_rank
		FROM client_orders
	)
	INSERT INTO reconciliation (
		order_id, client_id, trade_id, symbol, trade_date,
		matched_quantity, unmatched_quantity, status,
		avg_price, total_cost, execution_slippage
	)
	SELECT 
		c.order_id,
		c.client_id,
		b.trade_id,
		c.isin AS symbol,
		b.deal_date AS trade_date,
		CASE
			WHEN b.quantity = c.quantity THEN b.quantity  -- Exact match
			WHEN b.quantity < c.quantity THEN b.quantity  -- Partial match (allocate broker quantity)
			ELSE c.quantity  -- Partial match (allocate order quantity)
		END AS matched_quantity,
		ABS(b.quantity - c.quantity) AS unmatched_quantity,
		CASE
			WHEN b.quantity = c.quantity THEN 'MATCHED'
			WHEN b.quantity < c.quantity THEN 'PARTIAL'
			ELSE 'EXCESS'
		END AS status,
		(b.cost / b.quantity) * (CASE
			WHEN b.quantity < c.quantity THEN b.quantity
			ELSE c.quantity 
		END) AS avg_price,
		b.cost * (CASE
			WHEN b.quantity < c.quantity THEN b.quantity
			ELSE c.quantity 
		END) AS total_cost,
		ABS((b.cost / b.quantity) - (b.cost * (CASE
			WHEN b.quantity < c.quantity THEN b.quantity
			ELSE c.quantity 
		END) / b.quantity)) AS execution_slippage
	FROM broker_ranked b
	JOIN client_ranked c 
		ON b.isin = c.isin 
		AND b.deal_date = c.order_date 
		AND b.broker_rank = c.client_rank;`

	// Handle unmatched client orders
	unmatched_orders := `
	INSERT INTO reconciliation (
		order_id, client_id, trade_id, symbol, trade_date,
		matched_quantity, unmatched_quantity, status,
		avg_price, total_cost, execution_slippage
	)
	SELECT 
		order_id,
		client_id,
		NULL AS trade_id,
		isin AS symbol,
		order_date AS trade_date,
		0 AS matched_quantity,
		quantity AS unmatched_quantity,
		'PENDING' AS status,
		NULL AS avg_price,
		0 AS total_cost,
		0 AS execution_slippage
	FROM client_orders 
	WHERE order_id NOT IN (SELECT order_id FROM reconciliation);`

	// Handle excess broker trades
	excess_trades := `
	INSERT INTO reconciliation (
		order_id, client_id, trade_id, symbol, trade_date,
		matched_quantity, unmatched_quantity, status,
		avg_price, total_cost, execution_slippage
	)
	SELECT 
		NULL AS order_id,
		NULL AS client_id,
		trade_id,
		isin AS symbol,
		deal_date AS trade_date,
		0 AS matched_quantity,
		quantity AS unmatched_quantity,
		'EXCESS' AS status,
		(cost / quantity) AS avg_price,
		0 AS total_cost,
		0 AS execution_slippage
	FROM broker_trades 
	WHERE trade_id NOT IN (SELECT trade_id FROM reconciliation);`

	// Execute all queries within a transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	// Execute main reconciliation
	if _, err := tx.Exec(reconciliationQuery); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to execute main reconciliation: %v", err)
	}

	// Handle unmatched orders
	if _, err := tx.Exec(unmatched_orders); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to handle unmatched orders: %v", err)
	}

	// Handle excess trades
	if _, err := tx.Exec(excess_trades); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to handle excess trades: %v", err)
	}

	return tx.Commit()
}

func generateReports(db *sql.DB) error {
	// Query for matched trades report
	matchedQuery := `
	SELECT * FROM reconciliation 
	WHERE status IN ('MATCHED', 'PARTIAL')
	ORDER BY trade_date, symbol;`

	// Query for unmatched trades report
	unmatchedQuery := `
	SELECT * FROM reconciliation 
	WHERE status IN ('PENDING', 'EXCESS')
	ORDER BY trade_date, symbol;`

	// Query for broker summary
	brokerSummaryQuery := `
	SELECT 
		r.trade_date,
		b.broker_code,
		COUNT(*) as total_trades,
		SUM(CASE WHEN r.status = 'MATCHED' THEN 1 ELSE 0 END) as matched_trades,
		SUM(CASE WHEN r.status = 'PARTIAL' THEN 1 ELSE 0 END) as partial_trades,
		SUM(CASE WHEN r.status = 'EXCESS' THEN 1 ELSE 0 END) as excess_trades,
		AVG(r.execution_slippage) as avg_slippage
	FROM reconciliation r
	LEFT JOIN broker_trades b ON r.trade_id = b.trade_id
	WHERE b.broker_code IS NOT NULL
	GROUP BY r.trade_date, b.broker_code
	ORDER BY r.trade_date, b.broker_code;`

	// Generate reports using the queries above
	// (Report generation code remains similar to previous version)
	return nil
}

func main() {
	db, err := initDB("trades.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Process EML files and load data into database
	// (Email processing code remains similar to previous version)

	// Perform reconciliation
	if err := performReconciliation(db); err != nil {
		log.Fatalf("Failed to perform reconciliation: %v", err)
	}

	// Generate reports
	if err := generateReports(db); err != nil {
		log.Fatalf("Failed to generate reports: %v", err)
	}
}
