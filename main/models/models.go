package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TypeModel struct {
	ID        primitive.ObjectID
	StrField  string
	BoolField bool
	IntField  int
}
