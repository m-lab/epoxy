// Package datastorex extends the cloud.google.com/go/datastore package.
package datastorex

import (
	"cloud.google.com/go/datastore"
)

// TODO: If we move this package to github.com/m-lab/go, then we should change
// the Map type to be more general (i.e. map[string]interface{}) or change the
// type names to be more specific (i.e. StringMap).

// Map implements the PropertyLoadSaver interface and provides a mechanism for
// using `map[string]string` types in datastore entities.
type Map map[string]string

// Load allocates a new map and assignes key/values from each
// datastore.Property. Every datastore.Property type should be string.
// Non-string types are ignored.
func (mp *Map) Load(ps []datastore.Property) error {
	*mp = make(map[string]string)
	for _, p := range ps {
		s, ok := p.Value.(string)
		// Silently skip any types that are not strings.
		if ok {
			(*mp)[p.Name] = s
		}
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
			// Force property to not be indexed. This allows more freedom of the
			// property name, i.e. may contain ".".
			NoIndex: true,
		})
	}
	return d, nil
}
