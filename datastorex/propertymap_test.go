// Package datastorex extends the cloud.google.com/go/datastore package.
package datastorex

import (
	"reflect"
	"sort"
	"testing"

	"cloud.google.com/go/datastore"
)

func TestMap_Load(t *testing.T) {
	ps := []datastore.Property{
		{
			Name:    "a",
			Value:   "b",
			NoIndex: true,
		},
		{
			Name:    "c",
			Value:   "d",
			NoIndex: true,
		},
		{
			// Should be ignored.
			Name:  "i",
			Value: 100,
		},
	}

	m := Map{}
	if err := m.Load(ps); err != nil {
		t.Errorf("Map.Load() error = %v, want nil", err)
	}

	expected := Map{"a": "b", "c": "d"}
	if !reflect.DeepEqual(m, expected) {
		t.Errorf("Map.Load() got = %#v, want %#v", m, expected)
	}
}

func TestMap_Save(t *testing.T) {
	m := Map{"a": "b", "c": "d"}

	got, err := m.Save()
	if err != nil {
		t.Errorf("Map.Save() error = %v, want nil", err)
		return
	}

	sort.Slice(got, func(i, j int) bool {
		return got[i].Name < got[j].Name
	})

	expected := []datastore.Property{
		{
			Name:    "a",
			Value:   "b",
			NoIndex: true,
		},
		{
			Name:    "c",
			Value:   "d",
			NoIndex: true,
		},
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Map.Save() = %#v, want %#v", got, expected)
	}
}
