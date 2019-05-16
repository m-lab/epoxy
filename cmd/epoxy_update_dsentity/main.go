// Package main implements a simple utility to migrate

// This utility is a disposable tool that is only needed "once" to migrate an
// existing Datastore schema to a new schema. I say "once" because we may want
// to use this tool multiple times, but each schema change is a one-way step.
package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/m-lab/epoxy/datastorex"
	"github.com/m-lab/epoxy/storage"
	"src/github.com/m-lab/go/rtx"
)

var (
	fProject string
)

func init() {
	flag.StringVar(&fProject, "project", "mlab-sandbox", "GCP project name to update.")
}

type oldCollectedInformation struct {
	Platform         string
	BuildArch        string
	Serial           string
	Asset            string
	UUID             string
	Manufacturer     string
	Product          string
	Chip             string
	MAC              string
	IP               string
	Version          string
	PublicSSHHostKey string
}

// A oldHost represents the old datastore entity schema.
type oldHost struct {
	Name                 string
	IPv4Addr             string
	Boot                 storage.Sequence
	Update               storage.Sequence
	UpdateEnabled        bool
	Extensions           []string
	CurrentSessionIDs    storage.SessionIDs
	LastSessionCreation  time.Time
	LastReport           time.Time
	LastSuccess          time.Time
	CollectedInformation oldCollectedInformation
}

var oldEntityKind = "ePoxyHosts"

func oldList(c *datastore.Client) ([]*oldHost, error) {
	var hosts []*oldHost
	q := datastore.NewQuery(oldEntityKind)
	// Discard array of keys returned since we only need the values in hosts.
	_, err := c.GetAll(context.Background(), q, &hosts)
	if err != nil {
		return nil, err
	}
	return hosts, nil
}

func main() {
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	client, err := datastore.NewClient(ctx, fProject)
	rtx.Must(err, "Failed to create new datastore client")
	// Query old namespace and entities for all instances.
	oldHosts, err := oldList(client)
	rtx.Must(err, "Failed to list old Host entities")

	dsc := storage.NewDatastoreConfig(client)

	for _, old := range oldHosts {
		// For each one copy to a new storage.Host
		h := &storage.Host{
			Name:                old.Name,
			IPv4Addr:            old.IPv4Addr,
			Boot:                old.Boot,
			Update:              old.Update,
			UpdateEnabled:       old.UpdateEnabled,
			Extensions:          old.Extensions,
			CurrentSessionIDs:   old.CurrentSessionIDs,
			LastSessionCreation: old.LastSessionCreation,
			LastReport:          old.LastReport,
			LastSuccess:         old.LastSuccess,

			// We currently collect no information so we're not losing any.
			CollectedInformation: datastorex.Map{},
		}
		fmt.Println(h)
		// Save each one.
		dsc.Save(h)
	}
}
