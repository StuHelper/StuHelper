// Command migrate-sfzjh is a one-time migration tool that encrypts plaintext
// sfzjh data in academic.buaa_students and populates the sfzjh_hash column.
//
// After running migration 000016 (which renames sfzjh -> sfzjh_enc and converts
// to BYTEA via convert_to), the column contains raw UTF-8 bytes — not AES-GCM
// ciphertext. This tool reads each row, encrypts the value using the PII cipher,
// computes the HMAC-SHA256 hash, and updates both columns.
//
// Prerequisites:
//   - Migration 000016 has been applied
//   - Environment variables set: DATABASE_URL, DOC_AES_ACTIVE_KEY_ID, DOC_AES_KEYS, HMAC_SECRET
//
// Usage:
//
//	# From server/ directory (loads .env automatically):
//	go run ./cmd/migrate-sfzjh
//
//	# With explicit env:
//	DATABASE_URL=postgres://... DOC_AES_ACTIVE_KEY_ID=1 DOC_AES_KEYS=1:... HMAC_SECRET=... go run ./cmd/migrate-sfzjh
//
//	# Dry-run mode (read-only, no writes):
//	go run ./cmd/migrate-sfzjh --dry-run
package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto/pii"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/phoneutil"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "read-only mode: log what would be changed without writing")
	flag.Parse()

	// Load .env file if present (non-fatal if missing)
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("Warning: failed to load .env: %v", err)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	hmacSecret := os.Getenv("HMAC_SECRET")
	if hmacSecret == "" {
		log.Fatal("HMAC_SECRET environment variable is required")
	}
	hmacKey := []byte(hmacSecret)

	cipher, err := buildPIICipher()
	if err != nil {
		log.Fatalf("Failed to initialize PII cipher: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := run(ctx, pool, cipher, hmacKey, *dryRun); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
}

func run(ctx context.Context, pool *pgxpool.Pool, cipher *pii.Cipher, hmacKey []byte, dryRun bool) error {
	// Query all rows. We detect "not yet encrypted" rows by checking whether
	// the sfzjh_enc value starts with the PII envelope version byte (0x01).
	// Raw UTF-8 bytes from convert_to() will never start with 0x01 for valid
	// ID numbers (which start with ASCII digits or letters).
	rows, err := pool.Query(ctx, `
		SELECT xh, sfzjh_enc
		FROM academic.buaa_students
		WHERE sfzjh_enc IS NOT NULL
		  AND length(sfzjh_enc) > 0
	`)
	if err != nil {
		return fmt.Errorf("query students: %w", err)
	}
	defer rows.Close()

	type pending struct {
		xh        string
		plaintext string
	}
	var toMigrate []pending

	for rows.Next() {
		var xh string
		var sfzjhEnc []byte
		if err := rows.Scan(&xh, &sfzjhEnc); err != nil {
			return fmt.Errorf("scan row: %w", err)
		}

		// Skip rows that are already encrypted (envelope version 0x01)
		if len(sfzjhEnc) > 0 && sfzjhEnc[0] == 0x01 {
			log.Printf("[SKIP] xh=%s: already encrypted (envelope v1)", xh)
			continue
		}

		// The raw bytes are UTF-8 plaintext from convert_to()
		plaintext := strings.TrimSpace(string(sfzjhEnc))
		if plaintext == "" {
			log.Printf("[SKIP] xh=%s: empty plaintext after trim", xh)
			continue
		}

		toMigrate = append(toMigrate, pending{xh: xh, plaintext: plaintext})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate rows: %w", err)
	}

	if len(toMigrate) == 0 {
		log.Println("No rows need migration. All sfzjh_enc values are already encrypted or empty.")
		return nil
	}

	log.Printf("Found %d rows to migrate", len(toMigrate))

	if dryRun {
		for _, p := range toMigrate {
			hash, err := phoneutil.HashLookupWithKey(p.plaintext, hmacKey)
			if err != nil {
				return fmt.Errorf("hash xh=%s: %w", p.xh, err)
			}
			log.Printf("[DRY-RUN] xh=%s: would encrypt sfzjh_enc and set sfzjh_hash=%s...%s",
				p.xh, hash[:8], hash[len(hash)-8:])
		}
		log.Printf("[DRY-RUN] Would update %d rows. Re-run without --dry-run to apply.", len(toMigrate))
		return nil
	}

	var migrated, failed int
	for _, p := range toMigrate {
		encrypted, err := cipher.Encrypt(p.plaintext)
		if err != nil {
			log.Printf("[ERROR] xh=%s: encrypt failed: %v", p.xh, err)
			failed++
			continue
		}

		hash, err := phoneutil.HashLookupWithKey(p.plaintext, hmacKey)
		if err != nil {
			log.Printf("[ERROR] xh=%s: hash failed: %v", p.xh, err)
			failed++
			continue
		}

		_, err = pool.Exec(ctx, `
			UPDATE academic.buaa_students
			SET sfzjh_enc = $1, sfzjh_hash = $2
			WHERE xh = $3
		`, encrypted, hash, p.xh)
		if err != nil {
			log.Printf("[ERROR] xh=%s: update failed: %v", p.xh, err)
			failed++
			continue
		}

		migrated++
		log.Printf("[OK] xh=%s: encrypted and hashed", p.xh)
	}

	log.Printf("Migration complete: %d migrated, %d failed, %d total", migrated, failed, len(toMigrate))
	if failed > 0 {
		return fmt.Errorf("%d rows failed to migrate", failed)
	}
	return nil
}

// buildPIICipher constructs a PII cipher from DOC_AES_ACTIVE_KEY_ID and DOC_AES_KEYS
// environment variables. This mirrors the parsing logic in config/security.go.
func buildPIICipher() (*pii.Cipher, error) {
	activeKeyIDStr := os.Getenv("DOC_AES_ACTIVE_KEY_ID")
	if activeKeyIDStr == "" {
		return nil, fmt.Errorf("DOC_AES_ACTIVE_KEY_ID environment variable is required")
	}

	activeKeyID, err := strconv.ParseUint(activeKeyIDStr, 10, 8)
	if err != nil || activeKeyID < 1 || activeKeyID > 255 {
		return nil, fmt.Errorf("DOC_AES_ACTIVE_KEY_ID must be 1-255, got %q", activeKeyIDStr)
	}

	keysStr := os.Getenv("DOC_AES_KEYS")
	if keysStr == "" {
		return nil, fmt.Errorf("DOC_AES_KEYS environment variable is required")
	}

	keys := make(map[uint8][]byte)
	for _, entry := range strings.Split(keysStr, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("DOC_AES_KEYS: invalid entry %q, expected format keyID:hex", entry)
		}

		idStr := strings.TrimSpace(parts[0])
		hexStr := strings.TrimSpace(parts[1])

		id, err := strconv.ParseUint(idStr, 10, 8)
		if err != nil || id < 1 || id > 255 {
			return nil, fmt.Errorf("DOC_AES_KEYS: invalid key ID %q", idStr)
		}

		keyBytes, err := hex.DecodeString(hexStr)
		if err != nil {
			return nil, fmt.Errorf("DOC_AES_KEYS: invalid hex for key ID %d: %w", id, err)
		}
		if len(keyBytes) != 32 {
			return nil, fmt.Errorf("DOC_AES_KEYS: key ID %d must be exactly 32 bytes (got %d)", id, len(keyBytes))
		}

		keys[uint8(id)] = keyBytes
	}

	return pii.NewCipher(uint8(activeKeyID), keys)
}
