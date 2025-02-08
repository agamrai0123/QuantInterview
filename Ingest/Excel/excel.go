package Excel

import (
	"fmt"
	"log"
	"ingest/models"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

func ReadExcel(filename string) (*[]models.Trades, error) {
	var trades []models.Trades
	file, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	rows, err := file.GetRows("Sheet1")
	if err != nil {
		log.Fatal(err)
	}
	for i, row := range rows {
		if i == 0 {
			continue
		}

		if len(row) < 13 {
			continue // Skip rows with missing columns
		}

		// Parse each column correctly
		quantity, _ := strconv.ParseInt(strings.ReplaceAll(row[4], ",", ""), 10, 64)
		price, _ := strconv.ParseFloat(strings.ReplaceAll(row[5], ",", ""), 64)
		netAmount, _ := strconv.ParseFloat(strings.ReplaceAll(row[7], ",", ""), 64)
		brokerage, _ := strconv.ParseFloat(strings.ReplaceAll(row[8], ",", ""), 64)
		stt, _ := strconv.ParseInt(strings.ReplaceAll(row[10], ",", ""), 10, 32)

		trade := models.Trades{
			TradeDate:      row[0],
			BrokerCode:     row[1],
			ISIN:           row[2],
			BuySell:        rune(row[3][0]),
			Quantity:       quantity,
			Price:          price,
			NetAmount:      netAmount,
			Brokerage:      brokerage,
			SettlementDate: row[9],
			STT:            int32(stt),
			ExchangeCode:   row[11],
			Depository:     row[12],
		}

		trades = append(trades, trade)
	}
	fmt.Printf("Trade: %f\n", trades[0].NetAmount)
	return &trades, nil
}
