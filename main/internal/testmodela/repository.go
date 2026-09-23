package testmodela

import (
	"context"
	"fmt"

	"github.com/oraz200101/sandbox-server/main/internal/client"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	create = `INSERT INTO test_model_a (id, str_field, bool_field, int_field)
VALUES ($1, $2, $3, $4)
RETURNING id, str_field, bool_field, int_field`

	updateByID = `UPDATE test_model_a
SET str_field  = $2,
    bool_field = $3,
    int_field  = $4
WHERE id = $1
RETURNING id, str_field, bool_field, int_field`

	getByID = `SELECT id, str_field, bool_field, int_field
FROM test_model_a
WHERE id = $1`

	deleteByID = `DELETE FROM test_model_a
WHERE id = $1`
)

type Repository struct {
	client *client.Client
}

func NewRepository(client *client.Client) *Repository {
	return &Repository{client: client}
}

func (r *Repository) Create(ctx context.Context, testModelA *TestModelA) error {
	if _, err := r.client.PostgresClient.Db.ExecContext(
		ctx,
		create,
		testModelA.ID.Hex(),
		testModelA.StrField,
		testModelA.BoolField,
		testModelA.IntField,
	); err != nil {
		return fmt.Errorf("create test model: %w", err)
	}

	return nil
}

func (r *Repository) UpdateByID(ctx context.Context, testModelA *TestModelA) error {
	if _, err := r.client.PostgresClient.Db.ExecContext(
		ctx,
		updateByID,
		testModelA.ID.Hex(),
		testModelA.StrField,
		testModelA.BoolField,
		testModelA.IntField,
	); err != nil {
		return fmt.Errorf("update test model : %w", err)
	}

	return nil
}

func (r *Repository) GetByID(ctx context.Context, id bson.ObjectID) (*TestModelA, error) {
	var testModelA TestModelA

	err := r.client.PostgresClient.Db.QueryRowContext(ctx, getByID, id.Hex()).Scan(
		&testModelA.StrField,
		&testModelA.IntField,
		&testModelA.BoolField,
	)

}

func (r *Repository) DeleteByID(id string) error {
}

func (r *Repository) GetList() ([]TestModelA, error) {

}
