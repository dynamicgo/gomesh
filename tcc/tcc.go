package tcc

import "context"

// Session .
type Session interface {
	Context() context.Context
	Commit() error
	Rollback() error
}

// New .
func New(ctx context.Context) (Session, error) {
	return nil, nil
}
