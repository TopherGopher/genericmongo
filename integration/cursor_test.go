package integration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tophergopher/genericmongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// youngestTwo is a find option whose effect shows in both which documents
// come back and their order.
func youngestTwo() *options.FindOptions {
	return options.Find().SetSort(bson.M{"age": 1}).SetLimit(2)
}

func TestFindReturnsACursorOfDocuments(t *testing.T) {
	t.Parallel()
	ctx := deadline(t)
	users := usersWith(t, ada, bob, cy)

	cur, err := users.Find(ctx, bson.M{}, youngestTwo())
	require.NoError(t, err, "the find the test reads")
	defer cur.Close(ctx)

	var got []user
	for cur.Next(ctx) {
		u, err := cur.Decode()
		require.NoError(t, err, "each document decodes as a user")
		got = append(got, u)
	}
	require.NoError(t, cur.Err(), "the cursor must reach its end without an error")
	assert.Equal(t, []user{cy, ada}, got, "the cursor gives the documents the options chose, in their order")
}

func TestAllReturnsEveryDocument(t *testing.T) {
	t.Parallel()
	ctx := deadline(t)
	users := usersWith(t, ada, bob, cy)

	cur, err := users.Find(ctx, bson.M{}, youngestTwo())
	require.NoError(t, err, "the find the test reads")
	got, err := cur.All(ctx)
	require.NoError(t, err, "every document decodes as a user")
	assert.Equal(t, []user{cy, ada}, got, "All gives the documents the options chose, in their order")

	cur, err = users.Find(ctx, bson.M{"_id": "nobody"})
	require.NoError(t, err, "a find that matches nothing is no error")
	got, err = cur.All(ctx)
	require.NoError(t, err, "nothing to decode is no error")
	assert.Empty(t, got, "no documents match")
	assert.NotNil(t, got, "no documents is an empty slice, which JSON encodes as [] and not as null")
}

func TestAllFailureReturnsNoDocuments(t *testing.T) {
	t.Parallel()
	ctx := deadline(t)
	coll := misfiled(t)
	_, err := coll.InsertOne(ctx, bob)
	require.NoError(t, err, "a document that does decode, beside the one that does not")
	users := genericmongo.New[user](coll)

	// Bob sorts first, so he is decoded before the failure.
	cur, err := users.Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"_id": -1}))
	require.NoError(t, err, "the find itself must succeed")
	got, err := cur.All(ctx)

	var decodeErr *bsoncodec.DecodeError
	require.ErrorAs(t, err, &decodeErr, "a string cannot be decoded as an age")
	assert.Empty(t, got, "All returns no documents with an error, however it comes to read them")
}

func TestFindFailureReturnsNoCursor(t *testing.T) {
	t.Parallel()
	users := usersWith(t)

	cur, err := users.Find(deadline(t), bson.M{"$nope": 1})

	var refused mongo.CommandError
	assert.ErrorAs(t, err, &refused, "the server's refusal comes back as the driver's error")
	assert.Nil(t, cur, "no cursor comes with the error, as with the driver")
}

// ages is the shape of the aggregation's one result, which is not a user.
type ages struct {
	Total int `bson:"total"`
}

func TestNewCursorTypesAnAggregation(t *testing.T) {
	t.Parallel()
	ctx := deadline(t)
	users := usersWith(t, ada, bob, cy)
	sumOfAges := []bson.M{{"$group": bson.M{"_id": nil, "total": bson.M{"$sum": "$age"}}}}

	cur, err := users.Aggregate(ctx, sumOfAges)
	require.NoError(t, err, "Aggregate is the driver's")
	got, err := genericmongo.NewCursor[ages](cur).All(ctx)
	require.NoError(t, err, "the result decodes as the type the caller named")
	assert.Equal(t, []ages{{Total: ada.Age + bob.Age + cy.Age}}, got, "the driver's cursor is read as the caller's type")
}
