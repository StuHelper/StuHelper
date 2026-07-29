package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
)

func bootstrapSchoolTuples(ctx context.Context, apiURL, storeID, modelID string) error {
	schoolIDs, err := listSchoolIDs(ctx)
	if err != nil {
		return err
	}
	tuples := initialTuples(schoolIDs)
	if len(tuples) == 0 {
		log.Println("No schools found; skipped initial tuple write")
		return nil
	}

	client, err := fga.NewClient(config.OpenFGAConfig{
		APIUrl:               apiURL,
		StoreID:              storeID,
		AuthorizationModelID: modelID,
	})
	if err != nil {
		return fmt.Errorf("create tuple bootstrap client: %w", err)
	}
	if err := client.WriteMissingTuples(ctx, tuples); err != nil {
		return fmt.Errorf("write initial school tuples: %w", err)
	}
	log.Printf("Ensured %d initial tuples", len(tuples))
	return nil
}

func listSchoolIDs(ctx context.Context) ([]string, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(databaseURL) == "" {
		return nil, fmt.Errorf("DATABASE_URL is required to bootstrap OpenFGA school tuples")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	rows, err := pool.Query(ctx, `SELECT id::text FROM schools ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("query schools: %w", err)
	}
	defer rows.Close()

	var schoolIDs []string
	for rows.Next() {
		var schoolID string
		if err := rows.Scan(&schoolID); err != nil {
			return nil, fmt.Errorf("scan school id: %w", err)
		}
		schoolIDs = append(schoolIDs, schoolID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate school ids: %w", err)
	}
	return schoolIDs, nil
}

func initialTuples(schoolIDs []string) []fga.Tuple {
	tuples := make([]fga.Tuple, 0, len(schoolIDs)*2)
	for _, schoolID := range schoolIDs {
		if strings.TrimSpace(schoolID) == "" {
			continue
		}
		tuples = append(tuples,
			fga.Tuple{User: "ecosystem:stuhelper", Relation: "parent", Object: "school:" + schoolID},
			fga.Tuple{
				User:     "school:" + schoolID,
				Relation: "school",
				Object:   "section:" + fga.ReviewModerationSectionID(schoolID),
			},
		)
	}
	return tuples
}
