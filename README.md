# genericmongo

genericmongo wraps version 1 of the
[MongoDB Go driver](https://pkg.go.dev/go.mongodb.org/mongo-driver/mongo) so
that a read returns your own type and a write takes it.

```sh
go get github.com/tophergopher/genericmongo
```

It needs Go 1.26 or newer. Releases stay below v1.0.0 for now, so the API may
still change.

```go
type User struct {
    Name string `bson:"name"`
    Age  int    `bson:"age"`
}

// olderThan returns the users who are older than the one called name.
func olderThan(ctx context.Context, db *mongo.Database, name string) ([]User, error) {
    users := genericmongo.New[User](db.Collection("users"))

    named, err := users.FindOne(ctx, bson.M{"name": name}) // a User
    if err != nil {
        return nil, err
    }
    cursor, err := users.Find(ctx, bson.M{"age": bson.M{"$gt": named.Age}})
    if err != nil {
        return nil, err
    }
    return cursor.All(ctx) // a []User
}
```

A write takes the type too, so a slice needs no copy into a `[]interface{}`:

```go
_, err := users.InsertMany(ctx, []User{{Name: "Ada", Age: 36}, {Name: "Cy", Age: 12}})
```

## What changes from the driver

A `Collection[T]` embeds the driver's `*mongo.Collection`. Each method does
what the driver's method of the same name does, and takes the same filters
and options. These methods are typed:

| Method | With the driver | With genericmongo |
| --- | --- | --- |
| `FindOne`, `FindOneAndDelete`, `FindOneAndUpdate`, `FindOneAndReplace` | returns a result to decode | returns a `T` |
| `Find` | returns a cursor to decode | returns a `Cursor[T]`, whose `Decode` returns a `T` and `All` a `[]T` |
| `InsertOne`, `ReplaceOne`, `FindOneAndReplace` | takes the document as an `interface{}` | takes a `T` |
| `InsertMany` | takes a `[]interface{}` | takes a `[]T` |
| `Clone` | returns the driver's collection | returns a `Collection[T]` |

Results of writes are the driver's own, such as `*mongo.InsertOneResult` with
its `InsertedID`. So are errors: `errors.Is(err, mongo.ErrNoDocuments)`
matches as it does with the driver. A call that returns an error returns no
document with it: the value is the zero `T`.

Every other method is the driver's own and works unchanged: `UpdateOne`,
`DeleteOne`, `CountDocuments`, `Aggregate`, `Watch` and the rest. The
documents an aggregation returns are often not the collection's, so wrap its
cursor yourself:

```go
cur, err := users.Aggregate(ctx, pipeline)
if err != nil {
    return err
}
counts, err := genericmongo.NewCursor[CityCount](cur).All(ctx)
```

Code that needs the driver's own collection or cursor gets it from the
embedded field: `users.Collection`, `cursor.Cursor`.

`T` is whatever the collection holds: a struct, a pointer to one, or
`bson.M`. Your document type need not implement any interface, and the
package adds no query builder and no hooks.

## Testing code that uses a collection

The package exports no interface. To test your own code without a server,
declare an interface with the methods it calls. A `Collection[T]` has them,
and so can a stand-in you write for the test:

```go
type userFinder interface {
    FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) (User, error)
}
```

`Example_testDouble` in [`example_test.go`](example_test.go) shows one in use.

## Tests

The tests are in [`integration/`](integration/), a module of its own, so that
the library depends on the driver and nothing else. They run against a real
MongoDB that [dbtest](https://github.com/TopherGopher/dbtest) starts in
Docker, and fail without a Docker daemon.

```sh
go test -race ./...                      # the examples
(cd integration && go test -race ./...)  # the tests
```

Changes are held to the standard in [`AGENTS.md`](AGENTS.md).
