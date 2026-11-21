# genericmongo

genericmongo types the documents of a MongoDB collection. It wraps version 1
of the [MongoDB Go driver](https://pkg.go.dev/go.mongodb.org/mongo-driver/mongo),
so that a read returns your own type and a write takes it.

```sh
go get github.com/tophergopher/genericmongo
```

It needs Go 1.26 or newer.

```go
type User struct {
    Name string `bson:"name"`
    Age  int    `bson:"age"`
}

func adults(ctx context.Context, db *mongo.Database) ([]User, error) {
    users := genericmongo.New[User](db.Collection("users"))

    _, err := users.InsertMany(ctx, []User{{Name: "Ada", Age: 36}, {Name: "Cy", Age: 12}})
    if err != nil {
        return nil, err
    }

    ada, err := users.FindOne(ctx, bson.M{"name": "Ada"}) // ada is a User
    if err != nil {
        return nil, err
    }

    cursor, err := users.Find(ctx, bson.M{"age": bson.M{"$gte": ada.Age}})
    if err != nil {
        return nil, err
    }
    return cursor.All(ctx) // a []User
}
```

## What is typed

A `Collection[T]` is the driver's collection. Each method does what the
driver's method of the same name does, and takes the same filters and
options. Errors are the driver's own, so `errors.Is(err,
mongo.ErrNoDocuments)` matches as it does with the driver.

These methods take or return a `T` where the driver uses `interface{}`:

| Method | With the driver | With genericmongo |
| --- | --- | --- |
| `FindOne`, `FindOneAndDelete`, `FindOneAndUpdate`, `FindOneAndReplace` | return a result to decode | return a `T` |
| `Find` | returns a cursor to decode | returns a `Cursor[T]`, whose `Decode` and `All` return `T` and `[]T` |
| `InsertOne`, `ReplaceOne`, `FindOneAndReplace` | take an `interface{}` | take a `T` |
| `InsertMany` | takes a `[]interface{}` | takes a `[]T` |
| `Clone` | returns the driver's collection | returns a `Collection[T]` |

Every other method is the driver's own and works unchanged: `UpdateOne`,
`DeleteOne`, `CountDocuments`, `Aggregate`, `Watch` and the rest. The
documents an aggregation returns are seldom the collection's, so type its
cursor yourself:

```go
cur, err := users.Aggregate(ctx, pipeline)
if err != nil {
    return err
}
totals, err := genericmongo.NewCursor[Total](cur).All(ctx)
```

`T` is whatever the collection holds: a struct, a pointer to one, or
`bson.M`. A document type implements nothing, and the package adds no query
builder and no hooks.

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
