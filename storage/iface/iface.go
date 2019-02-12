package iface

import (
	"context"

	"cloud.google.com/go/datastore"
)

// DatastoreClient is an interface to make testing possible. The default
// implementation is the actual *datastore.Client as returned by
// datastore.NewClient.
type DatastoreClient interface {
	Get(ctx context.Context, key *datastore.Key, dst interface{}) error
	Put(ctx context.Context, key *datastore.Key, src interface{}) (*datastore.Key, error)
	GetAll(ctx context.Context, q *datastore.Query, dst interface{}) ([]*datastore.Key, error)
}
