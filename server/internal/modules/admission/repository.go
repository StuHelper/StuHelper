package admission

import (
	"context"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto/pii"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
)

type Repository struct {
	db            *db.DB
	authURLCipher pii.EncryptDecryptor
}

func NewRepository(database *db.DB, authURLCipher pii.EncryptDecryptor) *Repository {
	if database == nil {
		panic("admission.NewRepository: database must not be nil")
	}
	if authURLCipher == nil {
		panic("admission.NewRepository: authURLCipher must not be nil")
	}
	return &Repository{db: database, authURLCipher: authURLCipher}
}

// encryptAuthURL 把明文 join URL 加密为 at-rest 密文；空 URL 存 NULL（Encrypt 不接受空串）。
func (r *Repository) encryptAuthURL(authURL string) ([]byte, error) {
	if authURL == "" {
		return nil, nil
	}
	return r.authURLCipher.Encrypt(authURL)
}

// decryptAuthURL 把 at-rest 密文还原为明文 join URL；NULL/空字节还原为空串。
func (r *Repository) decryptAuthURL(enc []byte) (string, error) {
	if len(enc) == 0 {
		return "", nil
	}
	return r.authURLCipher.Decrypt(enc)
}

func withDBTable(ctx context.Context, table string) context.Context {
	return db.WithTableHint(ctx, table)
}

func (r *Repository) WithTx(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
	return r.db.WithTx(ctx, fn)
}
