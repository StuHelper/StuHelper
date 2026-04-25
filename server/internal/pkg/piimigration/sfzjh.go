package piimigration

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto/pii"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/phoneutil"
)

type PendingSfzjh struct {
	XH        string
	Plaintext string
}

func CollectPendingSfzjh(ctx context.Context, pool *pgxpool.Pool) ([]PendingSfzjh, error) {
	rows, err := pool.Query(ctx, `
		SELECT xh, sfzjh_enc
		FROM academic.buaa_students
		WHERE sfzjh_enc IS NOT NULL
		  AND length(sfzjh_enc) > 0
	`)
	if err != nil {
		return nil, fmt.Errorf("query students: %w", err)
	}
	defer rows.Close()

	pending := make([]PendingSfzjh, 0)
	for rows.Next() {
		var (
			xh       string
			sfzjhEnc []byte
		)
		if err := rows.Scan(&xh, &sfzjhEnc); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		if len(sfzjhEnc) > 0 && sfzjhEnc[0] == 0x01 {
			continue
		}
		plaintext := strings.TrimSpace(string(sfzjhEnc))
		if plaintext == "" {
			continue
		}
		pending = append(pending, PendingSfzjh{XH: xh, Plaintext: plaintext})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}
	return pending, nil
}

func ApplyPendingSfzjh(ctx context.Context, pool *pgxpool.Pool, pending []PendingSfzjh, cipher *pii.Cipher, hmacKey []byte, dryRun bool) error {
	if len(pending) == 0 {
		return nil
	}
	var failed int
	for _, item := range pending {
		hash, err := phoneutil.HashLookupWithKey(item.Plaintext, hmacKey)
		if err != nil {
			failed++
			continue
		}
		if dryRun {
			continue
		}
		encrypted, err := cipher.Encrypt(item.Plaintext)
		if err != nil {
			failed++
			continue
		}
		if _, err := pool.Exec(ctx, `
			UPDATE academic.buaa_students
			SET sfzjh_enc = $1, sfzjh_hash = $2
			WHERE xh = $3
		`, encrypted, hash, item.XH); err != nil {
			failed++
			continue
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d rows failed to migrate", failed)
	}
	return nil
}

func BuildCipherFromEnv() (*pii.Cipher, error) {
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

func LoadHMACKeyFromEnv() ([]byte, error) {
	hmacSecret := os.Getenv("HMAC_SECRET")
	if hmacSecret == "" {
		return nil, fmt.Errorf("HMAC_SECRET environment variable is required")
	}
	return []byte(hmacSecret), nil
}
