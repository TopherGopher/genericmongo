package genericmongo_test

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/tophergopher/genericmongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// User is the document the examples store.
type User struct {
	Name string `bson:"name"`
	Age  int    `bson:"age"`
}

// The examples that use a collection are compiled but not run, because they
// need a MongoDB server.

// New wraps a collection of the driver's. Writes then take the type, and
// reads return it.
func ExampleNew() {
	var db *mongo.Database // from the caller's client
	users := genericmongo.New[User](db.Collection("users"))

	ctx := context.Background()
	if _, err := users.InsertMany(ctx, []User{{Name: "Ada", Age: 36}, {Name: "Bob", Age: 52}}); err != nil {
		log.Fatal(err)
	}
	// UpdateOne is the driver's own method: its documents are not typed.
	if _, err := users.UpdateOne(ctx, bson.M{"name": "Ada"}, bson.M{"$inc": bson.M{"age": 1}}); err != nil {
		log.Fatal(err)
	}
}

// FindOne returns the document. When nothing matches, the error is
// mongo.ErrNoDocuments.
func ExampleCollection_FindOne() {
	var db *mongo.Database // from the caller's client
	users := genericmongo.New[User](db.Collection("users"))

	ada, err := users.FindOne(context.Background(), bson.M{"name": "Ada"})
	if errors.Is(err, mongo.ErrNoDocuments) {
		fmt.Println("nobody is called Ada")
		return
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(ada.Age)
}

// Find returns a cursor that is read as the driver's is, except that Decode
// returns the document.
func ExampleCollection_Find() {
	var db *mongo.Database // from the caller's client
	users := genericmongo.New[User](db.Collection("users"))

	ctx := context.Background()
	adults, err := users.Find(ctx, bson.M{"age": bson.M{"$gte": 18}}, options.Find().SetSort(bson.M{"name": 1}))
	if err != nil {
		log.Fatal(err)
	}
	defer adults.Close(ctx)
	for adults.Next(ctx) {
		adult, err := adults.Decode()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(adult.Name)
	}
	if err := adults.Err(); err != nil {
		log.Fatal(err)
	}
}

// All reads a cursor to its end and closes it.
func ExampleCursor_All() {
	var db *mongo.Database // from the caller's client
	users := genericmongo.New[User](db.Collection("users"))

	ctx := context.Background()
	cursor, err := users.Find(ctx, bson.M{"age": bson.M{"$gte": 18}})
	if err != nil {
		log.Fatal(err)
	}
	adults, err := cursor.All(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(adults))
}

// CityCount is one result of an aggregation that counts users by city.
type CityCount struct {
	City  string `bson:"_id"`
	Users int    `bson:"users"`
}

// An aggregation's documents are often not the collection's, so Aggregate
// stays the driver's and NewCursor wraps the cursor it returns. Here the
// driver builds that cursor from documents in memory, so the example runs
// with no server.
func ExampleNewCursor() {
	cursor, err := mongo.NewCursorFromDocuments([]any{
		bson.M{"_id": "London", "users": 2},
		bson.M{"_id": "Paris", "users": 1},
	}, nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	counts, err := genericmongo.NewCursor[CityCount](cursor).All(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	for _, count := range counts {
		fmt.Println(count.City, count.Users)
	}
	// Output:
	// London 2
	// Paris 1
}

// userFinder declares the one method that ageOf needs from a collection of
// users.
type userFinder interface {
	FindOne(ctx context.Context, filter any, opts ...*options.FindOneOptions) (User, error)
}

// A Collection[User] is a userFinder.
var _ userFinder = (*genericmongo.Collection[User])(nil)

// ageOf is the code under test: it reads a user and needs no server to be
// tested.
func ageOf(ctx context.Context, users userFinder, name string) (int, error) {
	user, err := users.FindOne(ctx, bson.M{"name": name})
	return user.Age, err
}

// onlyUser is a test's stand-in for a collection: it finds the same user
// whatever the filter.
type onlyUser User

func (u onlyUser) FindOne(context.Context, any, ...*options.FindOneOptions) (User, error) {
	return User(u), nil
}

// The package exports no interface. Code that wants a stand-in for a
// collection in its tests declares an interface with the methods it calls,
// which a Collection and the stand-in both have.
func Example_testDouble() {
	age, err := ageOf(context.Background(), onlyUser{Name: "Ada", Age: 36}, "Ada")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(age)
	// Output: 36
}
