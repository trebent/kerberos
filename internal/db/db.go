package db

import "context"

type SQLClient interface {
	ApplySchema(ctx context.Context, bs []byte) error
}
