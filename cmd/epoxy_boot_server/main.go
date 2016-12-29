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

// The ePoxy boot server is the first point of contact for managed machines as
// they boot. The boot server serves all client connections over TLS. And, the
// boot server restricts all state-changing requests to administrative users
// (any machine) and managed machines (only itself).
//
// Managed machines progress through three boot stages: 1) local boot media
// like an iPXE ROM, or an immutable CD image, 2) a minimal, linux-based
// network boot environment, 3) the final system image.
//
// Managed machines are treated as stateless. So, the ePoxy boot server acts as
// an external state manager that mediates the transition of successive boot
// stages. Managed machines positively acknowlege every stage transition using
// session ids generated on the first request and known only to the ePoxy boot
// server and the remote machine.
//
// So, if a managed machine acknowleges the final stage successfully, then we
// know that this machine is the same one that first contacted the ePoxy boot
// server.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

var (
	bindAddress = flag.String("hostname", "", "Listen for connections on this interface.")
	bindPort    = flag.Int("port", 8080, "Accept connections on this port.")
)

// addRoute adds a new handler function for a pattern-based URL target to a Gorilla mux.Router.
func addRoute(router *mux.Router, method, pattern string, handler http.HandlerFunc) {
	router.Methods(method).Path(pattern).Handler(http.Handler(handler))
}

// newRouter creates and initializes all routes for the ePoxy boot server.
func newRouter() *mux.Router {
	router := mux.NewRouter()

	// A health checker for running in Docker or AppEngine.
	addRoute(router, "GET", "/_ah/health", checkHealth)

	// Stage2 scripts are always the first script fetched by a booting machine.
	// "stage2.ipxe" is the target for ROM-based iPXE clients.
	addRoute(router, "POST", "/v1/boot/{hostname}/stage2.ipxe", generateStage2IPXE)

	// TODO(soltesz): add a target for CD-based ePoxy clients.
	// addRoute(router, "POST", "/v1/boot/{hostname}/stage2.json", generateStage2Json)

	// Next, begin, and end stage targets load after stage2 runs successfully.
	// TODO(soltesz): add targets for next, begin, and end stage targets.
	// addRoute(router, "POST", "/v1/boot/{hostname}/{sessionId}/nextstage.json", generateNextstage)
	// addRoute(router, "POST", "/v1/boot/{hostname}/{sessionId}/beginstage", handleBeginStage)
	// addRoute(router, "POST", "/v1/boot/{hostname}/{sessionId}/endstage", handleEndStage)

	// TODO(soltesz): add a target or retrieving all published SSH host keys.
	// addRoute(router, "GET", "/v1/boot/known_hosts", getKnownHosts)
	return router
}

// checkHealth reports whether the server is healthy. checkHealth will
// typically be registered as the http.Handler for the path "/_ah/health" when
// running in Docker or AppEngine.
func checkHealth(rw http.ResponseWriter, req *http.Request) {
	fmt.Fprint(rw, "ok")
}

// ipxeScriptTmpl contains the simplest iPXE script possible that simply runs
// the interactive iPXE shell and waits. This is a temporary stand-in for a
// script that will be generated at request time for a specific host. General
// documentation for ipxe scripts: http://ipxe.org/scripting
// TODO(soltesz): replace with a generic template.
const ipxeScriptTmpl = `#!ipxe
echo Booting %s
shell
`

// generateStage2IPXE creates the stage2 iPXE script for booting machines.
func generateStage2IPXE(rw http.ResponseWriter, req *http.Request) {
	hostname := mux.Vars(req)["hostname"]

	// TODO(soltesz):
	// * Use hostname as key to load record from Datastore.
	// * Verify that the source IP maches the host IP.
	// * Save information sent in PostForm.
	// * Generate new session IDs.
	// * Save host record to Datastore.
	// * Generate and send iPXE script with session IDs and the nextstage script from Datastore.
	rw.Header().Set("Content-Type", "text/plain; charset=us-ascii")
	rw.WriteHeader(http.StatusOK)
	_, err := fmt.Fprintf(rw, ipxeScriptTmpl, hostname)
	if err != nil {
		log.Printf("can't write response to %q: %v", hostname, err)
	}
}

func main() {
	// TODO(soltesz): support TLS natively for stand-alone mode. Though, this is not necessary for AppEngine.
	addr := fmt.Sprintf("%s:%d", *bindAddress, *bindPort)
	http.ListenAndServe(addr, newRouter())
}
