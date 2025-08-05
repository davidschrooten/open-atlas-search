package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/david/open-atlas-search/config"
)

// Client wraps MongoDB client with additional functionality
type Client struct {
	client   *mongo.Client
	database string
	timeout  time.Duration
}

// NewClient creates a new MongoDB client
func NewClient(cfg config.MongoDBConfig) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Timeout)*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(cfg.GetMongoURI())
	
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return &Client{
		client:   client,
		database: cfg.Database,
		timeout:  time.Duration(cfg.Timeout) * time.Second,
	}, nil
}

// Disconnect closes the MongoDB connection
func (c *Client) Disconnect() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	return c.client.Disconnect(ctx)
}

// Database returns the configured database
func (c *Client) Database() *mongo.Database {
	return c.client.Database(c.database)
}

// Collection returns a collection from the configured database
func (c *Client) Collection(name string) *mongo.Collection {
	return c.Database().Collection(name)
}

// FindDocuments retrieves documents from a collection with optional filter and projection
func (c *Client) FindDocuments(collection string, filter bson.M, limit int64) (*mongo.Cursor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	opts := options.Find()
	if limit > 0 {
		opts.SetLimit(limit)
	}
	
	// Optimize cursor for bulk operations
	opts.SetBatchSize(1000) // Fetch more documents per round trip
	opts.SetNoCursorTimeout(true) // Prevent cursor timeout for large datasets

	cursor, err := c.Collection(collection).Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find documents: %w", err)
	}

	return cursor, nil
}

// FindDocumentsSince finds documents modified since a given timestamp using a custom timestamp field
func (c *Client) FindDocumentsSince(collection, timestampField string, since time.Time, limit int64) (*mongo.Cursor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	var filter bson.M
	var sortField string

	if timestampField == "" || timestampField == "_id" {
		// Use ObjectID timestamp (default behavior)
		sinceObjectID := primitive.NewObjectIDFromTimestamp(since)
		filter = bson.M{"_id": bson.M{"$gt": sinceObjectID}}
		sortField = "_id"
	} else {
		// Use custom timestamp field
		filter = bson.M{timestampField: bson.M{"$gt": since}}
		sortField = timestampField
	}

	opts := options.Find().SetSort(bson.D{{Key: sortField, Value: 1}})
	if limit > 0 {
		opts.SetLimit(limit)
	}
	
	// Optimize cursor for incremental sync operations
	opts.SetBatchSize(500) // Smaller batch size for incremental updates
	opts.SetNoCursorTimeout(true)

	cursor, err := c.Collection(collection).Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find documents since %v: %w", since, err)
	}

	return cursor, nil
}

// GetLastDocumentTimestamp gets the timestamp of the most recent document using a custom timestamp field
func (c *Client) GetLastDocumentTimestamp(collection, timestampField string) (time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	var sortField string
	if timestampField == "" || timestampField == "_id" {
		sortField = "_id"
	} else {
		sortField = timestampField
	}

	opts := options.FindOne().SetSort(bson.D{{Key: sortField, Value: -1}})
	var result bson.M
	err := c.Collection(collection).FindOne(ctx, bson.M{}, opts).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return time.Time{}, nil // Return zero time if no documents
		}
		return time.Time{}, fmt.Errorf("failed to get last document: %w", err)
	}

	if timestampField == "" || timestampField == "_id" {
		// Use ObjectID timestamp
		if id, ok := result["_id"].(primitive.ObjectID); ok {
			return id.Timestamp(), nil
		}
		return time.Time{}, fmt.Errorf("document _id is not an ObjectID")
	} else {
		// Use custom timestamp field
		if timestamp, ok := result[timestampField]; ok {
		return c.ParseTimestamp(timestamp)
		}
		return time.Time{}, fmt.Errorf("timestamp field %s not found in document", timestampField)
	}
}

// ParseTimestamp parses various timestamp formats
func (c *Client) ParseTimestamp(timestamp interface{}) (time.Time, error) {
	switch t := timestamp.(type) {
	case time.Time:
		return t, nil
	case primitive.DateTime:
		return t.Time(), nil
	case int64:
		// Assume Unix timestamp
		return time.Unix(t, 0), nil
	case float64:
		// Assume Unix timestamp as float
		return time.Unix(int64(t), 0), nil
	case string:
		// Try to parse ISO 8601 format
		if parsedTime, err := time.Parse(time.RFC3339, t); err == nil {
			return parsedTime, nil
		}
		// Try to parse other common formats
		formats := []string{
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
		}
		for _, format := range formats {
			if parsedTime, err := time.Parse(format, t); err == nil {
				return parsedTime, nil
			}
		}
		return time.Time{}, fmt.Errorf("unable to parse timestamp string: %s", t)
	default:
		return time.Time{}, fmt.Errorf("unsupported timestamp type: %T", t)
	}
}

// CheckTimestampField checks if a timestamp field exists in the collection
func (c *Client) CheckTimestampField(collection, timestampField string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	if timestampField == "" || timestampField == "_id" {
		// _id always exists
		return true, nil
	}

	// Check if any document has this field
	filter := bson.M{timestampField: bson.M{"$exists": true}}
	count, err := c.Collection(collection).CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check timestamp field: %w", err)
	}

	return count > 0, nil
}

// AddTimestampField adds a timestamp field to all documents in a collection that don't have it
func (c *Client) AddTimestampField(collection, timestampField string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	if timestampField == "" || timestampField == "_id" {
		// No need to add _id field
		return nil
	}

	// Update documents that don't have the timestamp field
	filter := bson.M{timestampField: bson.M{"$exists": false}}
	update := bson.M{"$set": bson.M{timestampField: time.Now()}}

	result, err := c.Collection(collection).UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to add timestamp field: %w", err)
	}

	if result.ModifiedCount > 0 {
		fmt.Printf("Added %s field to %d documents in collection %s\n", timestampField, result.ModifiedCount, collection)
	}

	return nil
}

// GetCollectionStats returns statistics about a collection
func (c *Client) GetCollectionStats(collection string) (bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	var result bson.M
	err := c.Database().RunCommand(ctx, bson.D{
		{Key: "collStats", Value: collection},
	}).Decode(&result)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get collection stats: %w", err)
	}

	return result, nil
}

// CountDocuments returns the number of documents in a collection matching the filter
func (c *Client) CountDocuments(collection string, filter bson.M) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	count, err := c.Collection(collection).CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	return count, nil
}
