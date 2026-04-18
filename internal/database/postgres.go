// internal/database/postgres.go
package database

import (
    "database/sql"
    _ "github.com/lib/pq"
)

func ConnectPostgres(url string) (*sql.DB, error) {
    db, err := sql.Open("postgres", url)
    if err != nil {
        return nil, err
    }
    if err := db.Ping(); err != nil {
        return nil, err
    }
    return db, nil
}