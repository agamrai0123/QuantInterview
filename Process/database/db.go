package database

import (
	"database/sql"
)

func InitDB() (*sql.DB, error) {
	DB, err := sql.Open("sqlite", "./../trades.db")
	if err != nil {
		return nil, err
	}
	return DB, nil
}
