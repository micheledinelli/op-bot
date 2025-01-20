package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"op-bot/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const dbName = "op-bot-data"
const usersCollection = "users"
const chaptersCollection = "chapters"

type Mongo struct {
	Client *mongo.Client
	DbInfo DatabaseInfo
}

type DatabaseInfo struct {
	DatabaseName string
}

func InitDatabase(ctx context.Context, mongoUri string) (*Mongo, error) {
	opts := options.Client().ApplyURI(mongoUri)

	m := &Mongo{}

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, utils.ErrorDatabaseConnection
	}

	m.Client = client
	m.DbInfo.DatabaseName = dbName

	if err = client.Ping(ctx, nil); err != nil {
		return nil, utils.ErrorDatabasePing
	}

	return m, nil
}

func (m *Mongo) GetUsers() (*[]int64, error) {
	coll := m.Client.Database(
		m.DbInfo.DatabaseName,
	).Collection(usersCollection)

	var result bson.M

	cursor, err := coll.Find(context.TODO(), bson.D{{}})
	if err != nil {
		return nil, utils.ErrorMongoFind
	}

	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		if err := cursor.Decode(&result); err != nil {
			return nil, utils.ErrorMongoCursor
		}
	}

	if err := cursor.Err(); err != nil {
		return nil, utils.ErrorGenericMongoCursor
	}

	chatIds := make([]int64, 0)
	for k, v := range result {
		if k == "chat_id" {
			chatIds = append(chatIds, v.(int64))
		}
	}

	return &chatIds, nil
}

func (m *Mongo) AddUser(chatId int64) error {
	coll := m.Client.Database(
		m.DbInfo.DatabaseName,
	).Collection(usersCollection)

	var result bson.M

	coll.FindOne(
		context.TODO(),
		bson.D{{Key: "chat_id", Value: chatId}},
	).Decode(&result)

	// If the user is already registered we are done
	if result != nil {
		return nil
	}

	if _, err := coll.InsertOne(context.TODO(), bson.D{
		{Key: "chat_id", Value: chatId},
	}); err != nil {
		return utils.ErrorMongoInsertOne
	}

	return nil
}

func (m *Mongo) RemoveUser(chatId int64) error {
	coll := m.Client.Database(
		m.DbInfo.DatabaseName,
	).Collection(usersCollection)

	if _, err := coll.DeleteOne(context.TODO(),
		bson.D{{Key: "chat_id", Value: chatId}},
	); err != nil {
		return utils.ErrorMongoDeleteOne
	}

	return nil
}

func (m *Mongo) GetLatestChapter() (*utils.Chapter, error) {
	coll := m.Client.Database(
		m.DbInfo.DatabaseName,
	).Collection(chaptersCollection)

	chapter := &utils.Chapter{}

	if err := coll.FindOne(context.TODO(), bson.D{{}}).Decode(chapter); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("no latest chapter found: %w", err)
		}

		log.Println(err)
		return nil, utils.ErrorMongoFindOne
	}

	return chapter, nil
}

func (m *Mongo) UpdateLatestChapter(chapterNumber int64, url string) error {
	coll := m.Client.Database(
		m.DbInfo.DatabaseName,
	).Collection(chaptersCollection)

	filter := bson.D{{Key: "chapter_number", Value: chapterNumber}}

	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "chapter_number", Value: chapterNumber + 1},
		{Key: "latest_url", Value: url},
	}}}

	if _, err := coll.UpdateOne(context.TODO(), filter, update); err != nil {
		return utils.ErrorMongoUpdateOne
	}

	return nil
}
