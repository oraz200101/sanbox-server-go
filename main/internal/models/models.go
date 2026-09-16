package models

import "go.mongodb.org/mongo-driver/v2/bson"

type TypeModel struct {
	ID        bson.ObjectID
	StrField  string
	BoolField bool
	IntField  int
}
