package db

import "github.com/jackc/pgx/v5"

// Tx is the internal transaction handle shared across modules.
type Tx = pgx.Tx
