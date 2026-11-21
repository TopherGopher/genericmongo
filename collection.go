package genericmongo

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Collection wraps the driver's collection so that its methods take and
// return T. T is whatever the collection holds: a struct, a pointer to one,
// or bson.M.
//
// The methods declared here take or return a T. Every other method of
// [mongo.Collection], such as UpdateOne, DeleteOne, CountDocuments and
// Aggregate, is promoted from the embedded collection and works unchanged.
// Pass the embedded Collection to code that needs the driver's own.
type Collection[T any] struct {
	*mongo.Collection
}

// New wraps c so that its methods take and return T:
//
//	users := genericmongo.New[User](db.Collection("users"))
func New[T any](c *mongo.Collection) *Collection[T] {
	return &Collection[T]{Collection: c}
}

// FindOne returns the first document that matches filter. When none does, the
// error is [mongo.ErrNoDocuments].
func (c *Collection[T]) FindOne(ctx context.Context, filter any,
	opts ...*options.FindOneOptions) (T, error) {
	return decode[T](c.Collection.FindOne(ctx, filter, opts...).Decode)
}

// FindOneAndDelete deletes the first document that matches filter and returns
// it. When none does, the error is [mongo.ErrNoDocuments].
//
// The delete happens even when the document cannot then be decoded as T, and
// the document is not returned. To get the deleted document whatever its
// shape, call FindOneAndDelete on the embedded Collection: the driver's
// result holds the raw document.
func (c *Collection[T]) FindOneAndDelete(ctx context.Context, filter any,
	opts ...*options.FindOneAndDeleteOptions) (T, error) {
	return decode[T](c.Collection.FindOneAndDelete(ctx, filter, opts...).Decode)
}

// FindOneAndUpdate updates the first document that matches filter and returns
// it as it was before the update. To get it as it is after, pass
// SetReturnDocument(options.After). When there is no document to return, the
// error is [mongo.ErrNoDocuments]. As with FindOneAndDelete, the write happens
// even when the document cannot then be decoded as T.
func (c *Collection[T]) FindOneAndUpdate(ctx context.Context, filter, update any,
	opts ...*options.FindOneAndUpdateOptions) (T, error) {
	return decode[T](c.Collection.FindOneAndUpdate(ctx, filter, update, opts...).Decode)
}

// FindOneAndReplace replaces the first document that matches filter and
// returns it as it was before. To get the replacement, pass
// SetReturnDocument(options.After). When there is no document to return, the
// error is [mongo.ErrNoDocuments]. As with FindOneAndDelete, the write happens
// even when the document cannot then be decoded as T.
func (c *Collection[T]) FindOneAndReplace(ctx context.Context, filter any, replacement T,
	opts ...*options.FindOneAndReplaceOptions) (T, error) {
	return decode[T](c.Collection.FindOneAndReplace(ctx, filter, replacement, opts...).Decode)
}

// Find returns a cursor over the documents that match filter. Close the
// cursor when done with it; All closes it for you.
func (c *Collection[T]) Find(ctx context.Context, filter any,
	opts ...*options.FindOptions) (*Cursor[T], error) {
	cursor, err := c.Collection.Find(ctx, filter, opts...)
	if err != nil {
		return nil, err
	}
	return NewCursor[T](cursor), nil
}

// InsertOne inserts document.
func (c *Collection[T]) InsertOne(ctx context.Context, document T,
	opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	return c.Collection.InsertOne(ctx, document, opts...)
}

// InsertMany inserts documents.
func (c *Collection[T]) InsertMany(ctx context.Context, documents []T,
	opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error) {
	untyped := make([]any, len(documents))
	for i, document := range documents {
		untyped[i] = document
	}
	return c.Collection.InsertMany(ctx, untyped, opts...)
}

// ReplaceOne replaces the first document that matches filter with
// replacement.
func (c *Collection[T]) ReplaceOne(ctx context.Context, filter any, replacement T,
	opts ...*options.ReplaceOptions) (*mongo.UpdateResult, error) {
	return c.Collection.ReplaceOne(ctx, filter, replacement, opts...)
}

// Clone returns a copy of the collection with opts applied, for the same T.
func (c *Collection[T]) Clone(opts ...*options.CollectionOptions) (*Collection[T], error) {
	clone, err := c.Collection.Clone(opts...)
	if err != nil {
		return nil, err
	}
	return New[T](clone), nil
}
