package genericmongo

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

// Cursor wraps the driver's cursor so that Decode returns a T and All a []T.
// Every other method of [mongo.Cursor], such as Next, Err and Close, is
// promoted from the embedded cursor and works unchanged. Pass the embedded
// Cursor to code that needs the driver's own.
type Cursor[T any] struct {
	*mongo.Cursor
}

// NewCursor wraps c so that it decodes its documents as T. Use it on a cursor
// from one of the driver's own methods, such as Aggregate, whose documents
// are often not the collection's:
//
//	cur, err := users.Aggregate(ctx, pipeline)
//	if err != nil {
//		return err
//	}
//	totals, err := genericmongo.NewCursor[Total](cur).All(ctx)
func NewCursor[T any](c *mongo.Cursor) *Cursor[T] {
	return &Cursor[T]{Cursor: c}
}

// Decode returns the document that the last call to Next moved to.
func (c *Cursor[T]) Decode() (T, error) {
	return decode[T](c.Cursor.Decode)
}

// All returns the cursor's documents, an empty slice when there are none, and
// closes the cursor. Call it in place of Next, not after it: after Next, the
// driver returns the documents of the current batch again.
func (c *Cursor[T]) All(ctx context.Context) ([]T, error) {
	// Not nil, so that no documents encodes as a JSON array and not as null.
	documents := []T{}
	if err := c.Cursor.All(ctx, &documents); err != nil {
		return nil, err
	}
	return documents, nil
}

// decode returns as T the document that driverDecode reads, which is the
// Decode method of the driver's single result or of its cursor. A decode that
// fails can leave the document half filled, so the zero T is returned with
// the error.
func decode[T any](driverDecode func(any) error) (T, error) {
	var document T
	if err := driverDecode(&document); err != nil {
		var zero T
		return zero, err
	}
	return document, nil
}
