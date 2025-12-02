package repository

import (
	"database/sql"
)

type rdsRepository struct {
	db *sql.DB
}
