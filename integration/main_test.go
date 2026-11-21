package integration_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tophergopher/dbtest/mongotest"
	"github.com/tophergopher/genericmongo"
	"go.mongodb.org/mongo-driver/mongo"
)

// server is the one MongoDB every test in the package uses. Each test keeps
// to a collection of its own, so the tests run in parallel.
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

// user is the document type most tests store.
type user struct {
	ID   string `bson:"_id"`
	Name string `bson:"name"`
	Age  int    `bson:"age"`
}

// driverCollection returns an empty collection of the driver's, named after
// the test and dropped when the test ends.
func driverCollection(t *testing.T) *mongo.Collection {
	t.Helper()
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	coll := server.Database("genericmongo").Collection(name)
	// The test's context has ended by the time cleanup runs.
	t.Cleanup(func() { _ = coll.Drop(context.Background()) })
	return coll
}

// usersWith returns a typed collection for the test that holds the given
// users.
func usersWith(t *testing.T, seed ...user) *genericmongo.Collection[user] {
	t.Helper()
	users := genericmongo.New[user](driverCollection(t))
	if len(seed) > 0 {
		_, err := users.InsertMany(t.Context(), seed)
		require.NoError(t, err, "the users the test reads must be stored")
	}
	return users
}
