package genericmongo

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

// Cursor is the driver's cursor with its documents typed as T.
//
// Decode and All return T. Every other method of [mongo.Cursor], such as
// Next, Err and Close, is promoted from the embedded cursor and works
// unchanged.
type Cursor[T any] struct {
	*mongo.Cursor
}

// NewCursor returns c with its documents typed as T. Use it on a cursor from
// one of the driver's own methods, such as Aggregate, whose documents are
// often not the collection's:
//
//	cur, err := users.Aggregate(ctx, pipeline)
//	if err != nil {
//		return err
//	}
//	totals, err := genericmongo.NewCursor[Total](cur).All(ctx)
func NewCursor[T any](c *mongo.Cursor) *Cursor[T] {
	return &Cursor[T]{Cursor: c}
}

// Decode returns the document the last call to Next reached.
func (c *Cursor[T]) Decode() (T, error) {
	return decode[T](c.Cursor)
}

// All returns every document the cursor has left, and closes the cursor.
func (c *Cursor[T]) All(ctx context.Context) ([]T, error) {
	var documents []T
	if err := c.Cursor.All(ctx, &documents); err != nil {
		return nil, err
	}
	return documents, nil
}

// decoder is what the driver's single result and its cursor have in common.
type decoder interface {
	Decode(v any) error
}

// decode returns the document that from holds, as T. A decode that fails can
// leave the document half filled, so the zero T is returned with the error.
func decode[T any](from decoder) (T, error) {
	var document T
	if err := from.Decode(&document); err != nil {
		var zero T
		return zero, err
	}
	return document, nil
}
