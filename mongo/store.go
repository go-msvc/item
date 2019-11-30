package mongo

import (
	"context"
	"reflect"
	"time"

	"github.com/go-msvc/errors"
	"github.com/go-msvc/log"
	"github.com/go-msvc/store"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

//todo: limit nr of revisions kept
//todo: index on id, id+rev

func init() {
	store.Register("mongo", Config{})
}

//Config to make a mongo store
type Config struct {
	URI      string
	Database string
}

//Validate the config
func (c *Config) Validate() error {
	if len(c.URI) == 0 {
		c.URI = "mongodb://localhost:27017"
	}
	if len(c.Database) == 0 {
		return errors.Errorf("missing database name")
	}
	log.Debugf("Validated %T:%+v", c, *c)
	return nil
}

//New creates the mongo store
func (c Config) New(itemName string, itemType reflect.Type) (store.IStore, error) {
	if err := c.Validate(); err != nil {
		return nil, errors.Wrapf(err, "invalid config")
	}
	if err := store.ValidateUserType(itemType); err != nil {
		return nil, errors.Wrapf(err, "cannot store %v", itemType)
	}

	client, err := mongo.NewClient(options.Client().ApplyURI(c.URI))
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create mongo client to %s", c.URI)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to connect to mongo %s", c.URI)
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to check mongo %s", c.URI)
	}

	collection := client.Database(c.Database).Collection(itemName)

	//indexes:
	//	all docs implicitly have a unique _id (the mongo doc id)
	//	we add unique key on our item id and rev: id+rev

	//make unique index on "id"+"rev" combination to find old versions
	//(current version is not in this index because "id" is not set.
	// indexModel := mongo.IndexModel{
	// 	Keys: bson.M{
	// 		"id":  1, //index in ascending order (-1 for descending order)
	// 		"rev": 1, //index in ascending order (-1 for descending order)
	// 	},
	// 	Options: options.Index().SetUnique(true),
	// }
	// index, err := collection.Indexes().CreateOne(ctx, indexModel)

	// 	opts := options.CreateIndexes().SetMaxTime(10 * time.Second)
	// 	index := mongo.IndexModel{}
	// index.Keys = bsonx.Doc{{Key: *key, Value: bsonx.Int32(int32(*value))}}
	// if *unique {
	// 	index.Options = bsonx.Doc{{Key: "unique", Value: bsonx.Boolean(true)}}
	// }
	// collection.Indexes().CreateOne(context.Background(), index, opts)

	log.Debugf("Created mongo store(%s,%s,%s)", c.URI, c.Database, itemName)
	return &mongoStore{
		itemName:   itemName,
		itemType:   itemType,
		docType:    docType(itemType),
		collection: collection,
	}, nil
}

type mongoStore struct {
	itemName   string
	itemType   reflect.Type
	docType    reflect.Type
	collection *mongo.Collection
}

func (s mongoStore) Name() string {
	return s.itemName
}

func (s mongoStore) Type() reflect.Type {
	return s.itemType
}

func (s mongoStore) Add(v interface{}) (store.ItemInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info := store.ItemInfo{
		Rev:       1,
		Timestamp: time.Now().Truncate(time.Millisecond),
		//UserID: ...,
	}
	result, err := s.collection.InsertOne(
		ctx,
		bson.M{
			//"_id" is assigned by mongo
			"rev":  info.Rev,
			"id":   primitive.ObjectID{},
			"ts":   info.Timestamp,
			"user": primitive.ObjectID{},
			"data": v,
		})
	if err != nil {
		return store.ItemInfo{}, errors.Wrapf(err, "failed to insert into mongo")
	}

	oid, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return store.ItemInfo{}, errors.Wrapf(err, "failed to get inserted id")
	}

	info.ID = store.ID(oid.Hex())
	log.Debugf("Added %s:{id:\"%s\",rev:1}", s.itemName, info.ID)
	return info, nil
} //mongoStore.Add()

func (s mongoStore) Get(id store.ID) (interface{}, store.ItemInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, _ := primitive.ObjectIDFromHex(string(id))
	docPtrValue := reflect.New(s.docType)
	err := s.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(docPtrValue.Interface())
	if err != nil {
		return nil, store.ItemInfo{}, errors.Wrapf(err, "failed to get id=%s: %v", id, err)
	}
	docValue := docPtrValue.Elem()
	info := store.ItemInfo{
		ID:        store.ID(docValue.Field(IDFieldIndex).Interface().(primitive.ObjectID).Hex()),
		Rev:       docValue.Field(RevFieldIndex).Interface().(int),
		Timestamp: docValue.Field(TimestampFieldIndex).Interface().(time.Time),
		UserID:    store.ID(docValue.Field(UserIDFieldIndex).Interface().(primitive.ObjectID).Hex()),
	}
	log.Debugf("Got %s:{id:\"%s\",rev:%d}", s.itemName, info.ID, info.Rev)
	return docValue.Field(DataFieldIndex).Interface(), info, nil
} //mongoStore.Get()

func (s mongoStore) GetInfo(id store.ID) (store.ItemInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, _ := primitive.ObjectIDFromHex(string(id))
	head := docHead{}
	err := s.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&head)
	if err != nil {
		return store.ItemInfo{}, errors.Wrapf(err, "failed to get id=%s: %v", id, err)
	}

	info := store.ItemInfo{
		ID:        store.ID(head.ID.Hex()),
		Rev:       head.Rev,
		Timestamp: head.Timestamp,
		UserID:    store.ID(head.UserID.Hex()),
	}
	log.Debugf("Got info for %s:{id:\"%s\",rev:%d}", s.itemName, info.ID, info.Rev)
	return info, nil
} //mongoStore.GetRev()

func (s mongoStore) Upd(id store.ID, newData interface{}) (store.ItemInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//get current item with header info
	oldData, oldInfo, err := s.Get(id)
	if err != nil {
		return store.ItemInfo{}, errors.Wrapf(err, "cannot get item to upd")
	}

	objID, _ := primitive.ObjectIDFromHex(string(id))

	//make copy of old item
	{
		insertResult, err := s.collection.InsertOne(
			ctx,
			bson.M{
				//"_id": a new _id is assigned by mongo and is different from actual item id
				"rev":  oldInfo.Rev,
				"id":   objID, //store actual item id of current rev that becomes latest rev below
				"ts":   oldInfo.Timestamp,
				"user": oldInfo.UserID,
				"data": oldData,
			})
		if err != nil {
			return store.ItemInfo{}, errors.Wrapf(err, "failed to make copy of old item")
		}

		oid, ok := insertResult.InsertedID.(primitive.ObjectID)
		if !ok {
			return store.ItemInfo{}, errors.Wrapf(err, "failed to get inserted id of rev copy")
		}
		log.Debugf("Bak %s:{id:\"%s\",rev:%d} (mongo:_id:%s)", s.itemName, oldInfo.ID, oldInfo.Rev, oid)
	} //scope

	//now update existing doc with latest data and new rev nr
	newInfo := store.ItemInfo{
		ID:        oldInfo.ID,
		Rev:       oldInfo.Rev + 1,
		Timestamp: time.Now().Truncate(time.Millisecond),
		UserID:    "", //todo
	}
	_, err = s.collection.UpdateOne(ctx,
		bson.M{"_id": objID}, //update this existing doc
		bson.M{
			"$set": bson.M{
				//"_id" does not change
				"rev": newInfo.Rev,
				//"id": not set on latest revision, because not known when added, so keep consistent
				"ts":   newInfo.Timestamp,
				"user": newInfo.UserID,
				"data": newData,
			},
		})
	if err != nil {
		return store.ItemInfo{}, errors.Wrapf(err, "failed to upd rev=%d of id=%s: %v", newInfo.Rev, id, err)
	}
	log.Debugf("Upd %s:{id:\"%s\",rev:%d}", s.itemName, newInfo.ID, newInfo.Rev)
	return newInfo, nil
} //mongoStore.Upd()

func (s mongoStore) Del(id store.ID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, _ := primitive.ObjectIDFromHex(string(id))

	//delete the latest revision
	delResult, err := s.collection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		return errors.Wrapf(err, "failed to delete latest rev of id=%s: %v", id, err)
	}
	log.Debugf("Deleted %d documents for %s:{id:\"%s\"}", delResult.DeletedCount, s.itemName, id)

	//delete the older revisions
	delResult, err = s.collection.DeleteMany(ctx, bson.M{"id": objID})
	if err != nil {
		return errors.Wrapf(err, "failed to delete older rev of id=%s: %v", id, err)
	}
	log.Debugf("Deleted %d old documents for %s:{id:\"%s\"}", delResult.DeletedCount, s.itemName, id)

	return nil
} //mongoStore.Del()

// func (f factory) GetMsisdn(msisdn string) users.IUser {
// 	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
// 	defer cancel()

// 	cur, err := f.collection.Find(ctx, bson.M{"msisdn": msisdn})
// 	if err != nil {
// 		log.Errorf("Failed to find msisdn: %v", err)
// 		return nil
// 	}
// 	defer cur.Close(ctx)

// 	for cur.Next(ctx) {
// 		var result bson.M
// 		err := cur.Decode(&result)
// 		if err != nil {
// 			log.Errorf("Failed to get data: %v", err)
// 			return nil
// 		}
// 		// do something with result....
// 		log.Debugf("GOT (%T): %+v", result, result)
// 		u := mongoUser{
// 			id:       result["id"].(string),
// 			name:     result["name"].(string),
// 			msisdn:   result["msisdn"].(string),
// 			password: result["password"].(string),
// 		}
// 		return &u
// 	}
// 	if err := cur.Err(); err != nil {
// 		log.Errorf("Error: %v", err)
// 		return nil
// 	}
// 	return nil
// } //factory.GetMsisdn()

//docHead is present in each mongo document
//and a data field is added to each doc after this struct to store the user data
//the complete struct type is created in reflect with docType()
type docHead struct {
	ID        primitive.ObjectID `bson:"_id" doc:"Unique ID to access latest revision of the item."`
	Rev       int                `bson:"rev" doc:"Revision number"`
	ItemID    primitive.ObjectID `bson:"id" doc:"Item _id of the latest version (never changes)"`
	Timestamp time.Time          `bson:"ts" doc:"Timestamp when this revision was created."`
	UserID    primitive.ObjectID `bson:"user-id" doc:"User _id who created this."`
	//Data follows but not part of head
}

//docType() is the complete struct type of each mongo doc
//it is same as struct{docHead, data:<user type>}:
//
// field bson-tag  type           description
//	[0]  "_id"     mongo.ObjectID is unique for each document and used for item id
//	[1]  "rev"     int            is revision nr 1,2,3,...
//	[2]  "item-id" string         is set when preserving old revision, copying the _id if the original item
//	[3]  "ts"      time.Time      is the timestamp when this revision was created
//	[4]  "user-id" string         is the user _id
//	[5]  "data"    <user type>    is the data struct stored for user data of this item
const (
	//IDFieldIndex ...
	IDFieldIndex = 0
	//RevFieldIndex ...
	RevFieldIndex = 1
	//ItemIDFieldIndex ...
	ItemIDFieldIndex = 2
	//TimestampFieldIndex ...
	TimestampFieldIndex = 3
	//UserIDFieldIndex ...
	UserIDFieldIndex = 4
	//DataFieldIndex ...
	DataFieldIndex = 5

	//todo: also add user and timestamp for each rev...
	//todo: limit nr of rev that is kept (default: unlimited)
	//todo: user data key combinations
)

//docType makes a struct type that we store for each doc in the db
//containing _id followed by the user struct
func docType(userType reflect.Type) reflect.Type {
	fields := make([]reflect.StructField, 0)
	//docHead part:
	docHeadType := reflect.TypeOf(docHead{})
	for i := 0; i < docHeadType.NumField(); i++ {
		f := docHeadType.Field(i)
		fields = append(fields, reflect.StructField{
			Name: f.Name,
			Type: f.Type,
			Tag:  f.Tag,
		})
	}
	//data part:
	fields = append(fields, reflect.StructField{
		Name: "Data",
		Type: userType,
		Tag:  reflect.StructTag(`bson:"data"`),
	})
	return reflect.StructOf(fields)
} //docType()
