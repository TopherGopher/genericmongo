// Package genericmongo wraps version 1 of the MongoDB Go driver,
// go.mongodb.org/mongo-driver, so that a read returns a caller's own type and
// a write takes it:
//
//	users := genericmongo.New[User](db.Collection("users"))
//	ada, err := users.FindOne(ctx, bson.M{"name": "Ada"})
//
// A [Collection] embeds the driver's collection. Each method does what the
// driver's method of the same name does, with the same filters and options;
// the ones that a document goes into or comes out of take or return a T. The
// driver's other methods, such as UpdateOne and Aggregate, are there
// unchanged. [NewCursor] wraps the cursor that one of them returns.
//
// Errors are the driver's own, so errors.Is(err, mongo.ErrNoDocuments) matches
// as it does with the driver. A call that returns an error returns no
// document with it: the value is the zero T.
package genericmongo
