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
	"fmt"
	"reflect"
	"testing"

	"cloud.google.com/go/datastore"
)

// fakeDatastoreClient implements the datastoreClient interface for testing.
// Every operation should be successful.
type fakeDatastoreClient struct {
	host *Host
}

// Get reads the Host value from f.host and copies it to dst.
func (f *fakeDatastoreClient) Get(ctx context.Context, key *datastore.Key, dst interface{}) error {
	// Copy the host from f.host into dst.
	h, ok := dst.(*Host)
	if !ok {
		return fmt.Errorf("type assertion failed: got %T; want *Host", dst)
	}
	*h = *f.host
	return nil
}

// Put reads the Host value from src and copies it to f.host.
func (f *fakeDatastoreClient) Put(ctx context.Context, key *datastore.Key, src interface{}) (*datastore.Key, error) {
	// Copy the host from src into f.host.
	h, ok := src.(*Host)
	if !ok {
		return nil, fmt.Errorf("type assertion failed: got %T; want *Host", src)
	}
	*f.host = *h
	return nil, nil
}

func (f *fakeDatastoreClient) GetAll(ctx context.Context, q *datastore.Query, dst interface{}) ([]*datastore.Key, error) {
	// Extract the pointer to a list of *Host, and append f.host to the list.
	hosts, ok := dst.(*[]*Host)
	if !ok {
		return nil, fmt.Errorf("type assertion failed: got %T; want *[]*Host", dst)
	}
	*hosts = append(*hosts, f.host)
	return nil, nil
}

// errDatastoreClient implements a datastoreClient interface where every call fails with an error.
// The error returned is defined in errDatastoreClient.err.
type errDatastoreClient struct {
	err error
}

func (f *errDatastoreClient) Get(ctx context.Context, key *datastore.Key, dst interface{}) error {
	return f.err
}

func (f *errDatastoreClient) Put(ctx context.Context, key *datastore.Key, src interface{}) (*datastore.Key, error) {
	return nil, f.err
}

func (f *errDatastoreClient) GetAll(ctx context.Context, q *datastore.Query, dst interface{}) ([]*datastore.Key, error) {
	return nil, f.err
}

func TestNewDatastoreClient(t *testing.T) {
	h := Host{
		Name: "mlab1.iad1t.measurement-lab.org",
	}
	f := &fakeDatastoreClient{&h}
	c := NewDatastoreConfig(f)

	h2, err := c.Load("malb1.iad1t.measurement-lab.org")
	if err != nil {
		t.Fatal(err)
	}
	if h.Name != h2.Name {
		t.Errorf("Load for NewDatastoreConfig failed; want %q, got %q", h.Name, h2.Name)
	}
}

func TestDatastore(t *testing.T) {
	// NB: we store a partial Host record for brevity.
	h := Host{
		Name:     "mlab1.iad1t.measurement-lab.org",
		IPv4Addr: "165.117.240.9",
		Boot: Sequence{
			Stage1ChainURL: "https://example.com/path/stage1to2/stage1to2.ipxe",
		},
		CurrentSessionIDs: SessionIDs{
			Stage2ID: "01234",
		},
	}
	// Declare the fake datastore client outside the function below so we can access member elements.
	f := &fakeDatastoreClient{&h}
	c := &DatastoreConfig{f}

	// Store host record.
	err := c.Save(&h)
	if err != nil {
		t.Fatalf("Failed to save host: %s", err)
	}
	if !reflect.DeepEqual(&h, f.host) {
		t.Fatalf("Host records does not match: got %#v; want %#v\n", f.host, &h)
	}

	// Retrieve host record.
	h2, err := c.Load("mlab1.iad1t.measurement-lab.org")
	if err != nil {
		t.Fatalf("Failed to load host: %s", err)
	}
	if !reflect.DeepEqual(&h, h2) {
		t.Fatalf("Host records does not match: got %#v; want %#v\n", h2, &h)
	}

	// GetAll all hosts.
	hosts, err := c.List()
	if err != nil {
		t.Fatalf("Failed to list hosts: %s", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("Failed to list hosts: got %d; want 1\n", len(hosts))
	}
}

func TestDatastoreFailures(t *testing.T) {
	// NB: we store a partial Host record for brevity.
	h := Host{
		Name:     "mlab1.iad1t.measurement-lab.org",
		IPv4Addr: "165.117.240.9",
		Boot: Sequence{
			Stage1ChainURL: "https://example.com/path/stage1to2/stage1to2.ipxe",
		},
		CurrentSessionIDs: SessionIDs{
			Stage2ID: "01234",
		},
	}
	// Declare the fake datastore client outside the function below so we can access member elements.
	f := &errDatastoreClient{fmt.Errorf("Fake failure")}
	c := &DatastoreConfig{f}

	// Store host record.
	err := c.Save(&h)
	if err != f.err {
		t.Fatalf("Saved without error: got %q; want %q\n", err, f.err)
	}

	// Retrieve host record.
	_, err = c.Load("mlab1.iad1t.measurement-lab.org")
	if err != f.err {
		t.Fatalf("Load without error: got %q; want %q\n", err, f.err)
	}

	// GetAll all hosts.
	_, err = c.List()
	if err != f.err {
		t.Fatalf("List without error: got %q; want %q\n", err, f.err)
	}
}
