package tradeRecon

import "database/sql"

func TradeReconcillation(DB *sql.DB) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}

	query := `WITH broker_ranked AS (
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
				        WHEN b.quantity = c.quantity THEN b.quantity
				        WHEN b.quantity < c.quantity THEN b.quantity
				        ELSE c.quantity
				    END AS matched_quantity,
				    ABS(b.quantity - c.quantity) AS unmatched_quantity,
				    CASE
				        WHEN b.quantity = c.quantity THEN 'MATCHED'
				        WHEN b.quantity < c.quantity THEN 'PARTIAL'
				        ELSE 'EXCESS'
				    END AS status,
				    (b.cost / b.quantity) AS avg_price,
				    b.cost * (CASE
				        WHEN b.quantity < c.quantity THEN b.quantity
				        ELSE c.quantity 
				    END) / b.quantity AS total_cost,
				    ABS(c.order_price - (b.cost / b.quantity)) AS execution_slippage
				FROM broker_ranked b
				JOIN client_ranked c 
				    ON b.isin = c.isin 
				    AND b.deal_date = c.order_date 
				    AND b.broker_rank = c.client_rank`

	return tx.Commit()
}
