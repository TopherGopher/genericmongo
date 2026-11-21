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
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

// oldestFirst sorts by age, so a test can show an option reached the driver
// by which document came back.
var oldestFirst = bson.M{"age": -1}

func TestInsertOneAndFindOne(t *testing.T) {
	t.Parallel()
	ctx := deadline(t)
	users := usersWith(t)

	res, err := users.InsertOne(ctx, ada)
	require.NoError(t, err, "the insert the test reads back")
	assert.Equal(t, ada.ID, res.InsertedID, "the result is the driver's, holding the document's own _id")

	got, err := users.FindOne(ctx, bson.M{"_id": ada.ID})
	require.NoError(t, err, "the inserted document must be found")
	assert.Equal(t, ada, got, "FindOne returns the document that was inserted")

	_, err = users.InsertOne(ctx, ada)
	assert.True(t, mongo.IsDuplicateKeyError(err), "the driver's duplicate-key error comes back as it is")
}

func TestFindOneTakesOptions(t *testing.T) {
	t.Parallel()
	users := usersWith(t, ada, bob, cy)

	got, err := users.FindOne(deadline(t), bson.M{}, options.FindOne().SetSort(oldestFirst))
	require.NoError(t, err, "one of the three users must be found")
	assert.Equal(t, bob, got, "the sort option chooses which document FindOne returns")
}

func TestInsertManyTakesASliceOfDocuments(t *testing.T) {
	t.Parallel()
	ctx := deadline(t)
	users := usersWith(t)

	res, err := users.InsertMany(ctx, []user{ada, bob, cy})
	require.NoError(t, err, "the insert the test counts")
	assert.Len(t, res.InsertedIDs, 3, "every document in the slice is inserted")

	_, err = users.InsertMany(ctx, []user{ada, {ID: "dee"}}, options.InsertMany().SetOrdered(false))
	assert.True(t, mongo.IsDuplicateKeyError(err), "the driver's duplicate-key error comes back as it is")
	n, err := users.CountDocuments(ctx, bson.M{})
	require.NoError(t, err, "the count must run")
	assert.EqualValues(t, 4, n, "an unordered insert stores the documents after the one that failed")

	_, err = users.InsertMany(ctx, nil)
	assert.ErrorIs(t, err, mongo.ErrEmptySlice, "the driver's error for nothing to insert comes back as it is")
}

func TestReplaceOne(t *testing.T) {
	t.Parallel()
	ctx := deadline(t)
	users := usersWith(t, ada)
	renamed := user{ID: ada.ID, Name: "Countess", Age: ada.Age}

	res, err := users.ReplaceOne(ctx, bson.M{"_id": ada.ID}, renamed)
	require.NoError(t, err, "the replacement the test reads back")
	assert.EqualValues(t, 1, res.MatchedCount, "the result is the driver's")
	got, err := users.FindOne(ctx, bson.M{"_id": ada.ID})
	require.NoError(t, err, "the replaced document must be found")
	assert.Equal(t, renamed, got, "the stored document is the replacement")

	res, err = users.ReplaceOne(ctx, bson.M{"_id": bob.ID}, bob, options.Replace().SetUpsert(true))
	require.NoError(t, err, "a replacement that matches nothing is no error")
	assert.EqualValues(t, 1, res.UpsertedCount, "the upsert option inserts the replacement")
}

func TestFindOneAndUpdate(t *testing.T) {
	t.Parallel()
	ctx := deadline(t)
	users := usersWith(t, ada)
	birthday := bson.M{"$inc": bson.M{"age": 1}}

	before, err := users.FindOneAndUpdate(ctx, bson.M{"_id": ada.ID}, birthday)
	require.NoError(t, err, "the user must be found and updated")
	assert.Equal(t, ada, before, "the document as it was before the update, as the driver returns")

	after, err := users.FindOneAndUpdate(ctx, bson.M{"_id": ada.ID}, birthday,
		options.FindOneAndUpdate().SetReturnDocument(options.After))
	require.NoError(t, err, "the user must be found and updated again")
	assert.Equal(t, ada.Age+2, after.Age, "the return-document option gives the document after both updates")
}

func TestFindOneAndReplace(t *testing.T) {
	t.Parallel()
	ctx := deadline(t)
	users := usersWith(t, ada)
	renamed := user{ID: ada.ID, Name: "Countess", Age: ada.Age}

	before, err := users.FindOneAndReplace(ctx, bson.M{"_id": ada.ID}, renamed)
	require.NoError(t, err, "the user must be found and replaced")
	assert.Equal(t, ada, before, "the document as it was before the replacement, as the driver returns")

	after, err := users.FindOneAndReplace(ctx, bson.M{"_id": ada.ID}, ada,
		options.FindOneAndReplace().SetReturnDocument(options.After))
	require.NoError(t, err, "the user must be found and replaced again")
	assert.Equal(t, ada, after, "the return-document option gives the replacement")
}

func TestFindOneAndDelete(t *testing.T) {
	t.Parallel()
	ctx := deadline(t)
	users := usersWith(t, ada, bob, cy)

	got, err := users.FindOneAndDelete(ctx, bson.M{}, options.FindOneAndDelete().SetSort(oldestFirst))
	require.NoError(t, err, "one of the three users must be found and deleted")
	assert.Equal(t, bob, got, "the sort option chooses which document is deleted and returned")

	_, err = users.FindOne(ctx, bson.M{"_id": bob.ID})
	assert.ErrorIs(t, err, mongo.ErrNoDocuments, "the returned document is gone")
}

func TestNoMatchIsTheDriversError(t *testing.T) {
	t.Parallel()
	ctx := deadline(t)
	users := usersWith(t, ada)
	nobody := bson.M{"_id": "nobody"}

	reads := map[string]func() (user, error){
		"FindOne":          func() (user, error) { return users.FindOne(ctx, nobody) },
		"FindOneAndDelete": func() (user, error) { return users.FindOneAndDelete(ctx, nobody) },
		"FindOneAndUpdate": func() (user, error) {
			return users.FindOneAndUpdate(ctx, nobody, bson.M{"$set": bson.M{"age": 1}})
		},
		"FindOneAndReplace": func() (user, error) { return users.FindOneAndReplace(ctx, nobody, bob) },
	}
	for name, read := range reads {
		t.Run(name, func(t *testing.T) {
			got, err := read()
			assert.ErrorIs(t, err, mongo.ErrNoDocuments, "a caller's check for the driver's error still matches")
			assert.Zero(t, got, "no document comes with the error")
		})
	}
}

func TestDecodeFailureReturnsTheZeroValue(t *testing.T) {
	t.Parallel()
	anyone := bson.M{}

	reads := map[string]func(*testing.T, *genericmongo.Collection[user]) (user, error){
		"FindOne": func(t *testing.T, users *genericmongo.Collection[user]) (user, error) {
			return users.FindOne(deadline(t), anyone)
		},
		"FindOneAndDelete": func(t *testing.T, users *genericmongo.Collection[user]) (user, error) {
			return users.FindOneAndDelete(deadline(t), anyone)
		},
		"FindOneAndUpdate": func(t *testing.T, users *genericmongo.Collection[user]) (user, error) {
			return users.FindOneAndUpdate(deadline(t), anyone, bson.M{"$set": bson.M{"name": "Eve"}})
		},
		"FindOneAndReplace": func(t *testing.T, users *genericmongo.Collection[user]) (user, error) {
			return users.FindOneAndReplace(deadline(t), anyone, ada)
		},
		"Cursor.Decode": func(t *testing.T, users *genericmongo.Collection[user]) (user, error) {
			cur, err := users.Find(deadline(t), anyone)
			require.NoError(t, err, "the find itself must succeed")
			defer cur.Close(deadline(t))
			require.True(t, cur.Next(deadline(t)), "the cursor must reach the document")
			return cur.Decode()
		},
	}
	for name, read := range reads {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := read(t, genericmongo.New[user](misfiled(t)))

			var decodeErr *bsoncodec.DecodeError
			assert.ErrorAs(t, err, &decodeErr, "the driver's decode error comes back as it is")
			assert.Zero(t, got, "a document that failed to decode is not returned half filled")
		})
	}
}

func TestDecodeFailureReturnsANilPointer(t *testing.T) {
	t.Parallel()
	ctx := deadline(t)
	users := genericmongo.New[*user](misfiled(t))

	got, err := users.FindOne(ctx, bson.M{})
	require.Error(t, err, "a string cannot be decoded as an age")
	assert.Nil(t, got, "FindOne gives no pointer to a half-filled document")

	cur, err := users.Find(ctx, bson.M{})
	require.NoError(t, err, "the find itself must succeed")
	defer cur.Close(ctx)
	require.True(t, cur.Next(ctx), "the cursor must reach the document")
	got, err = cur.Decode()
	require.Error(t, err, "a string cannot be decoded as an age")
	assert.Nil(t, got, "the cursor gives no pointer to a half-filled document")
}

func TestPointerDocuments(t *testing.T) {
	t.Parallel()
	ctx := deadline(t)
	users := genericmongo.New[*user](driverCollection(t))
	want := ada

	_, err := users.InsertOne(ctx, &want)
	require.NoError(t, err, "the insert the test reads back")

	got, err := users.FindOne(ctx, bson.M{"_id": want.ID})
	require.NoError(t, err, "the inserted document must be found")
	assert.Equal(t, &want, got, "a collection of pointers returns a pointer to the document")

	got, err = users.FindOne(ctx, bson.M{"_id": "nobody"})
	assert.ErrorIs(t, err, mongo.ErrNoDocuments, "no document matches")
	assert.Nil(t, got, "no pointer comes with the error")
}

func TestMapDocuments(t *testing.T) {
	t.Parallel()
	ctx := deadline(t)
	docs := genericmongo.New[bson.M](driverCollection(t))
	want := bson.M{"_id": "ada", "name": "Ada"}

	_, err := docs.InsertOne(ctx, want)
	require.NoError(t, err, "the insert the test reads back")

	got, err := docs.FindOne(ctx, bson.M{"_id": "ada"})
	require.NoError(t, err, "the inserted document must be found")
	assert.Equal(t, want, got, "a collection of maps returns the document as a map")
}

func TestCloneKeepsTheTypeAndTakesOptions(t *testing.T) {
	t.Parallel()
	ctx := deadline(t)
	users := usersWith(t, ada)

	clone, err := users.Clone(options.Collection().SetWriteConcern(writeconcern.Unacknowledged()))
	require.NoError(t, err, "the clone the test uses")

	got, err := clone.FindOne(ctx, bson.M{"_id": ada.ID})
	require.NoError(t, err, "the clone reads the same collection")
	assert.Equal(t, ada, got, "the clone's documents have the original's type")

	_, err = clone.InsertOne(ctx, bob)
	assert.ErrorIs(t, err, mongo.ErrUnacknowledgedWrite, "the clone has the options it was given")
	_, err = users.InsertOne(ctx, cy)
	assert.NoError(t, err, "the original keeps its own options")
}

func TestDriverMethodsAreUnchanged(t *testing.T) {
	t.Parallel()
	ctx := deadline(t)
	users := usersWith(t, ada, bob, cy)

	updated, err := users.UpdateOne(ctx, bson.M{"_id": ada.ID}, bson.M{"$set": bson.M{"age": 37}})
	require.NoError(t, err, "UpdateOne is the driver's")
	assert.EqualValues(t, 1, updated.ModifiedCount, "one user is a year older")

	deleted, err := users.DeleteOne(ctx, bson.M{"_id": cy.ID})
	require.NoError(t, err, "DeleteOne is the driver's")
	assert.EqualValues(t, 1, deleted.DeletedCount, "one user is deleted")

	n, err := users.CountDocuments(ctx, bson.M{})
	require.NoError(t, err, "CountDocuments is the driver's")
	assert.EqualValues(t, 2, n, "two users are left")
}
