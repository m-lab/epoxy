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
	"github.com/m-lab/go/rtx"
)

var (
	project string
)

func init() {
	flag.StringVar(&project, "project", "mlab-sandbox", "GCP project name to update.")
}

type oldSequence struct {
	Stage1ChainURL string
	Stage2ChainURL string
	Stage3ChainURL string
}

// A oldHost represents the old datastore entity schema.
type oldHost struct {
	Name                 string
	IPv4Addr             string
	Boot                 oldSequence
	Update               oldSequence
	UpdateEnabled        bool
	Extensions           []string
	CurrentSessionIDs    storage.SessionIDs
	LastSessionCreation  time.Time
	LastReport           time.Time
	LastSuccess          time.Time
	CollectedInformation datastorex.Map
}

var oldEntityKind = "Host"
var oldNamespace = "ePoxy"

func oldList(c *datastore.Client) ([]*oldHost, error) {
	var hosts []*oldHost
	q := datastore.NewQuery(oldEntityKind).Namespace(oldNamespace)
	// Discard array of keys returned since we only need the values in hosts.
	_, err := c.GetAll(context.Background(), q, &hosts)
	if err != nil {
		return nil, err
	}
	return hosts, nil
}

func translateSequence(s oldSequence, stage string) datastorex.Map {
	stage1IPXE := fmt.Sprintf("https://epoxy-boot-api.%s.measurementlab.net:4430/v1/storage/%s/stage1to2.ipxe", project, stage)
	stage1JSON := fmt.Sprintf("https://storage.googleapis.com/epoxy-%s/%s/stage1to2.json", project, stage)
	return datastorex.Map{
		storage.Stage1IPXE: stage1IPXE,
		storage.Stage1JSON: stage1JSON,
		storage.Stage2:     s.Stage2ChainURL,
		storage.Stage3:     s.Stage3ChainURL,
	}
}

func main() {
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	client, err := datastore.NewClient(ctx, project)
	rtx.Must(err, "Failed to create new datastore client")
	// Query old namespace and entities for all instances.
	oldHosts, err := oldList(client)
	rtx.Must(err, "Failed to list old Host entities")

	dsc := storage.NewDatastoreConfig(client)

	for _, old := range oldHosts {
		// For each one copy to a new storage.Host
		h := &storage.Host{
			Name:                 old.Name,
			IPv4Addr:             old.IPv4Addr,
			Boot:                 translateSequence(old.Boot, "stage3_ubuntu"),
			Update:               translateSequence(old.Update, "stage3_mlxupdate"),
			UpdateEnabled:        old.UpdateEnabled,
			Extensions:           old.Extensions,
			CurrentSessionIDs:    old.CurrentSessionIDs,
			LastSessionCreation:  old.LastSessionCreation,
			LastReport:           old.LastReport,
			LastSuccess:          old.LastSuccess,
			CollectedInformation: old.CollectedInformation,
		}
		fmt.Println(h)
		// Save each one.
		dsc.Save(h)
	}
}
