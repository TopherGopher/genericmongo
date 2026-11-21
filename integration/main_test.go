package integration_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tophergopher/dbtest/mongotest"
	"github.com/tophergopher/genericmongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// server is the one MongoDB every test in the package uses. Each test keeps
// to collections of its own, so the tests run in parallel.
var server *mongotest.Instance

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

// run starts MongoDB, runs the tests and removes the container. A start that
// fails, a missing Docker daemon included, fails the run rather than skipping
// it.
func run(m *testing.M) int {
	ctx := context.Background()
	inst, err := mongotest.Start(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot start MongoDB: %v\n", err)
		return 1
	}
	server = inst
	code := m.Run()
	if err := inst.Stop(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "cannot remove MongoDB: %v\n", err)
		return 1
	}
	return code
}

// user is the document most tests store.
type user struct {
	ID   string `bson:"_id"`
	Name string `bson:"name"`
	Age  int    `bson:"age"`
}

var (
	ada = user{ID: "ada", Name: "Ada", Age: 36}
	bob = user{ID: "bob", Name: "Bob", Age: 52}
	cy  = user{ID: "cy", Name: "Cy", Age: 19}
)

// deadline returns a context that ends with the test or after a minute, so a
// call that hangs fails its own test and not the whole run.
func deadline(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	t.Cleanup(cancel)
	return ctx
}

// collections numbers the collections the tests make, so that no two share a
// name, even when a test runs twice in one process.
var collections atomic.Int64

// driverCollection returns an empty collection of the driver's for one test,
// dropped when the test ends.
func driverCollection(t *testing.T) *mongo.Collection {
	t.Helper()
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	coll := server.Database("genericmongo").Collection(fmt.Sprintf("%s_%d", name, collections.Add(1)))
	// The test's context has ended by the time cleanup runs.
	t.Cleanup(func() { _ = coll.Drop(context.Background()) })
	return coll
}

// usersWith returns a typed collection for one test, holding the given users.
func usersWith(t *testing.T, seed ...user) *genericmongo.Collection[user] {
	t.Helper()
	users := genericmongo.New[user](driverCollection(t))
	if len(seed) > 0 {
		_, err := users.InsertMany(deadline(t), seed)
		require.NoError(t, err, "the users the test reads must be stored")
	}
	return users
}

// misfiled returns a collection whose one document cannot be decoded as a
// user, because its age is a string. Its _id and name come before the age, so
// a decoder has filled them in by the time it fails.
func misfiled(t *testing.T) *mongo.Collection {
	t.Helper()
	coll := driverCollection(t)
	_, err := coll.InsertOne(deadline(t), bson.D{
		{Key: "_id", Value: "ada"},
		{Key: "name", Value: "Ada"},
		{Key: "age", Value: "thirty-six"},
	})
	require.NoError(t, err, "the document the test fails to decode must be stored")
	return coll
}
