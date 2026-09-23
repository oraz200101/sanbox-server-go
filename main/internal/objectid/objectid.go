package objectid

import (
	"database/sql/driver"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type ObjectID bson.ObjectID

func (id *ObjectID) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("ObjectID: expected string, got %T", src)
	}
	oid, err := bson.ObjectIDFromHex(s)
	if err != nil {
		return err
	}
	*id = ObjectID(oid)
	return nil
}

func (id ObjectID) Value() (driver.Value, error) {
	return bson.ObjectID(id).Hex(), nil
}
