package hdwallet

import (
	"context"

	"github.com/google/uuid"
)

type Generator interface {
	GenerateAddress(uuid uuid.UUID) (string, error)
}

type Collector interface {
	Collect(ctx context.Context, address string, uuids []uuid.UUID) error
}
