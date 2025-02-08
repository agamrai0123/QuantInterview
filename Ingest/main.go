package main

import (
	"fmt"
	"ingest/Excel"
	"ingest/database"

	_ "modernc.org/sqlite"
)

func main() {
	DB, err := database.InitDB()
	if err != nil {
		panic(err.Error())
	}
	defer DB.Close()

	orders, err := database.ReadClientOrders(DB)
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(orders)

	filename := "TradeFile1.xlsx"

	trades, err := Excel.ReadExcel(filename)
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Trade: %+v\n", trades)

	// err = database.InsertBrokerTrades(DB, *trades)
	// if err != nil {
	// 	panic(err.Error())
	// }

	// fmt.Println("Trade saved successfully.")
}
