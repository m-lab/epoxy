// Package datastorex extends the cloud.google.com/go/datastore package.
package datastorex

import (
	"cloud.google.com/go/datastore"
)

// Map implements the PropertyLoadSaver interface and provides a mechanism for
// using `map[string]string` types in datastore entities.
type Map map[string]string

// Load allocates a new map and assignes key/values from each
// datastore.Property. Every datastore.Property type must be string.
func (mp *Map) Load(ps []datastore.Property) error {
	*mp = make(map[string]string)
	m := *mp
	for _, p := range ps {
		m[p.Name] = p.Value.(string)
	}
	return nil
}

// Save converts a Map to a slice of datastore properties.
func (mp *Map) Save() ([]datastore.Property, error) {
	var d []datastore.Property
	for k, v := range *mp {
		d = append(d, datastore.Property{
			Name:  k,
			Value: v,
		})
	}
	return d, nil
}
