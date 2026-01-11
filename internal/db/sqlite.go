package db

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func MustOpen(dataSourceName string) *sql.DB {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		panic(err)
	}
	return db
}
