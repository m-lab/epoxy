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

// Package handler provides functions for responding to specific client
// requests by the ePoxy boot server.
package handler

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/m-lab/epoxy/storage"
	"github.com/m-lab/epoxy/template"
)

// Config provides access to Host records.
type Config interface {
	Save(host *storage.Host) error
	Load(name string) (*storage.Host, error)
}

// Env holds data necessary for executing handler functions.
type Env struct {
	// Config provides access to Host records.
	Config Config
	// ServerAddr is the host:port of the public service. Used to generate absolute URLs.
	ServerAddr string
}

// Handler objects satisfy the http.Handler interface. A Handler contains an environment
// for executing the associated handler function.
type Handler struct {
	*Env
	// HandlerFunc handles a request using the included Env.
	HandlerFunc func(env *Env, rw http.ResponseWriter, req *http.Request) (int, error)
}

// ServeHTTP satisfies the http.Handler interface.
func (h Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	code, err := h.HandlerFunc(h.Env, rw, req)
	if err != nil {
		http.Error(rw, err.Error(), code)
	}
}

// GenerateStage1IPXE creates the stage1 iPXE script for booting machines.
func GenerateStage1IPXE(env *Env, rw http.ResponseWriter, req *http.Request) (int, error) {
	hostname := mux.Vars(req)["hostname"]

	// Use hostname as key to load record from Datastore.
	host, err := env.Config.Load(hostname)
	if err != nil {
		return http.StatusNotFound, err
	}
	// TODO(soltesz):
	// * Verify that the source IP maches the host IP.
	// * Save information sent in PostForm.

	// Generate new session IDs.
	if err := host.GenerateSessionIDs(); err != nil {
		return http.StatusInternalServerError, err
	}

	// Save host record to Datastore to commit session IDs.
	if err := env.Config.Save(host); err != nil {
		return http.StatusInternalServerError, err
	}

	// Generate iPXE script.
	script, err := template.FormatStage1IPXEScript(host, env.ServerAddr)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Complete request as successful.
	rw.Header().Set("Content-Type", "text/plain; charset=us-ascii")
	rw.WriteHeader(http.StatusOK)
	_, err = fmt.Fprintf(rw, script)
	if err != nil {
		log.Printf("Failed to write response to %q: %v", hostname, err)
	}
	return http.StatusOK, nil
}
