package customer

import (
	"time"

	"github.com/oraz200101/sandbox-server/main/internal/charm"
	"github.com/oraz200101/sandbox-server/main/internal/objectid"
)

type Customer struct {
	ID         objectid.ObjectID
	Name       string
	Surname    string
	Patronymic string
	Charm      *charm.Charm
	Gender     Gender
	BirthDate  time.Time
}

type Gender string

const (
	GenderMale   Gender = "MALE"
	GenderFemale Gender = "FEMALE"
)
