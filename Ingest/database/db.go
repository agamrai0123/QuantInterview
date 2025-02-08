package database

import (
	"database/sql"
	"fmt"
	"ingest/models"
	"time"
)

func InitDB() (*sql.DB, error) {
	DB, err := sql.Open("sqlite", "./../trades.db")
	if err != nil {
		return nil, err
	}
	return DB, nil
}

func ReadClientOrders(DB *sql.DB) (*[]models.Orders, error) {
	var orders []models.Orders

	rows, err := DB.Query(`SELECT * from client_orders`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var order models.Orders
		if err := rows.Scan(&order.OrderId, &order.ClientId, &order.ISIN, &order.Quantity, &order.BuySell, &order.OrderDate); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return &orders, nil
}

func InsertBrokerTrades(DB *sql.DB, trades []models.Trades) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`INSERT INTO TradeData (deal_date, party_code, isin, buy_sell, 
							quantity, cost, net_amount, brokerage, settlement_date, 
							stt, exchange_code, depository_code) 
							VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, trade := range trades {
		_, err := stmt.Exec(formatDate(trade.TradeDate), trade.BrokerCode, trade.ISIN, string(trade.BuySell),
			trade.Quantity, trade.Price, trade.NetAmount, trade.Brokerage, formatDate(trade.SettlementDate),
			trade.STT, trade.ExchangeCode, trade.Depository)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func formatDate(InputDate string) string {

	t, err := time.Parse("01-02-06", InputDate)
	if err != nil {
		fmt.Println("Error parsing date:", err)
	}

	// Convert to YYYY-MM-DD format
	formattedDate := t.Format("2006-01-02")

	return formattedDate
}
