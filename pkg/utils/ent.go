package utils

import (
	"context"
	"momoko/internal/data/ent/gen"
)

func WithTx(ctx context.Context, client *gen.Client, fn func(tx *gen.Tx) error) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	if err = fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
