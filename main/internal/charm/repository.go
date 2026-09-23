package charm

import (
	"github.com/oraz200101/sandbox-server/main/internal/client"
)

type Repository struct {
	client *client.Client
}

func NewRepository(client *client.Client) *Repository {
	return &Repository{client: client}
}
