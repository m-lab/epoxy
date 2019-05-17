// Copyright 2016 ePoxy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//////////////////////////////////////////////////////////////////////////////

package storage

import (
	"context"

	"cloud.google.com/go/datastore"
	"github.com/m-lab/epoxy/storage/iface"
)

const (
	// entityKind categorizes the Datastore records.
	entityKind = "Host"
	// namespace places all epoxy entities in a unique Datastore namespace.
	namespace = "ePoxy"
)

// DatastoreConfig contains configuration for accessing Google Cloud Datastore.
type DatastoreConfig struct {
	Client iface.DatastoreClient
	// Kind is the datastore entity kind, e.g. "Host".
	Kind string
	// Namespace is the datastore namespace for all storage operations.
	Namespace string
}

// NewDatastoreConfig creates a new DatastoreConfig instance from a *datastore.Client.
func NewDatastoreConfig(client iface.DatastoreClient) *DatastoreConfig {
	return &DatastoreConfig{
		Client:    client,
		Kind:      entityKind,
		Namespace: namespace,
	}
}

// Load retrieves a Host record from the datastore.
func (c *DatastoreConfig) Load(name string) (*Host, error) {
	h := &Host{}
	key := datastore.NameKey(c.Kind, name, nil)
	key.Namespace = c.Namespace
	if err := c.Client.Get(context.Background(), key, h); err != nil {
		return nil, err
	}
	return h, nil
}

// Save stores a Host record to Datastore. Host names are globally unique. If
// a Host record already exists, then it is overwritten.
func (c *DatastoreConfig) Save(host *Host) error {
	key := datastore.NameKey(c.Kind, host.Name, nil)
	key.Namespace = c.Namespace
	if _, err := c.Client.Put(context.Background(), key, host); err != nil {
		return err
	}
	return nil
}

// List retrieves all Host records currently in the Datastore.
// TODO(soltesz): support some simple query filtering or subsets.
func (c *DatastoreConfig) List() ([]*Host, error) {
	var hosts []*Host
	q := datastore.NewQuery(c.Kind).Namespace(c.Namespace)
	// Discard array of keys returned since we only need the values in hosts.
	_, err := c.Client.GetAll(context.Background(), q, &hosts)
	if err != nil {
		return nil, err
	}
	return hosts, nil
}
