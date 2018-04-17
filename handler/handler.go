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
	"path"
	"time"

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

// GenerateStage1IPXE creates the stage1 iPXE script for booting machines.
// func (env *Env) GenerateStage1IPXE(rw http.ResponseWriter, req *http.Request) (int, error) {
func (env *Env) GenerateStage1IPXE(rw http.ResponseWriter, req *http.Request) {
	hostname := mux.Vars(req)["hostname"]

	// Use hostname as key to load record from Datastore.
	host, err := env.Config.Load(hostname)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}
	// TODO(soltesz):
	// * Verify that the source IP maches the host IP.
	// * Save information sent in PostForm.

	// Generate new session IDs.
	host.GenerateSessionIDs()

	// Save host record to Datastore to commit session IDs.
	if err := env.Config.Save(host); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate iPXE script.
	script := template.FormatStage1IPXEScript(host, env.ServerAddr)

	// Complete request as successful.
	rw.Header().Set("Content-Type", "text/plain; charset=us-ascii")
	rw.WriteHeader(http.StatusOK)
	_, err = fmt.Fprintf(rw, script)
	if err != nil {
		log.Printf("Failed to write response to %q: %v", hostname, err)
	}
	return
}

// GenerateJSONConfig creates and returns a JSON serialized nextboot.Config
// suitable for responding to stage2 or stage3 requests.
func (env *Env) GenerateJSONConfig(rw http.ResponseWriter, req *http.Request) {
	hostname := mux.Vars(req)["hostname"]
	// TODO: Verify that the sessionID matches the host.CurrentSessionIDs.Stage2ID.
	// sessionID := mux.Vars(req)["sessionID"]

	// Use hostname as key to load record from Datastore.
	host, err := env.Config.Load(hostname)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}
	// TODO(soltesz):
	// * Verify that the source IP maches the host IP.
	// * Save information sent in PostForm, e.g. ssh host key.
	stage := path.Base(req.URL.Path)

	script := template.FormatJSONConfig(host, stage)

	// Complete request as successful.
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
	_, err = fmt.Fprintf(rw, script)
	if err != nil {
		log.Printf("Failed to write response to %q: %v", hostname, err)
	}
	return
}

// ReceiveReport handles the last step of a boot sequence when the epoxy client reports
// success or failure. In both cases, the session ids are invalidated. In all cases,
// epoxy_client is expected to report the server's public host key.
func (env *Env) ReceiveReport(rw http.ResponseWriter, req *http.Request) {
	// TODO: log or save values where appropriate.
	req.ParseForm()

	// Use hostname as key to load record from Datastore.
	hostname := mux.Vars(req)["hostname"]
	host, err := env.Config.Load(hostname)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusNotFound)
		return
	}

	// Verify sessionID matches the host record (i.e. request is authorized).
	sessionID := mux.Vars(req)["sessionID"]
	if sessionID != host.CurrentSessionIDs.ReportID {
		http.Error(rw, "Given session ID does not match host record", http.StatusForbidden)
		return
	}

	host.LastReport = time.Now()
	status := req.PostForm.Get("message")
	if status == "success" {
		// When the status is success, disable the "update" and mark the time.
		host.LastSuccess = host.LastReport
		host.UpdateEnabled = false
		// TODO: invalidate session ids.
	}

	// Save the new host state.
	if err := env.Config.Save(host); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: log using structured JSON.
	log.Println(req.PostForm)

	// Report success with no content.
	rw.WriteHeader(http.StatusNoContent)
	return
}
