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
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/datastore"
	"github.com/gorilla/mux"
	"github.com/m-lab/epoxy/handler"
	"github.com/m-lab/epoxy/storage"
	"golang.org/x/net/context"
)

var (
	// Environment variables are preferred to flags for deployments in
	// AppEngine. And, using environment variables is encouraged for
	// twelve-factor apps -- https://12factor.net/config

	// projectID must be set using the GCLOUD_PROJECT environment variable.
	projectID = os.Getenv("GCLOUD_PROJECT")

	// publicAddr must be set if *not* running in AppEngine. When running in
	// AppEngine publicAddr is set automatically from a combination of
	// GCLOUD_PROJECT and GAE_SERVICE environment variables. When not running in
	// AppEngine, or to override the default in AppEngine, the PUBLIC_ADDRESS
	// environment variable must be set instead.
	publicAddr = os.Getenv("PUBLIC_ADDRESS")

	// bindAddress may be set using the LISTEN environment variable. By default, ePoxy
	// listens on all available interfaces.
	bindAddress = os.Getenv("LISTEN")

	// bindPort may be set using the PORT environment variable.
	bindPort = "8080"
)

// init checks the environment for configuration values.
func init() {
	// Only use the automatic public address if PUBLIC_ADDRESS is not already set.
	if service := os.Getenv("GAE_SERVICE"); service != "" && projectID != "" && publicAddr == "" {
		publicAddr = fmt.Sprintf("%s-dot-%s.appspot.com", service, projectID)
	}
	if port := os.Getenv("PORT"); port != "" {
		bindPort = port
	}
}

// addRoute adds a new handler for a pattern-based URL target to a Gorilla mux.Router.
func addRoute(router *mux.Router, method, pattern string, handler http.Handler) {
	router.Methods(method).Path(pattern).Handler(handler)
}

// newRouter creates and initializes all routes for the ePoxy boot server.
func newRouter(env *handler.Env) *mux.Router {
	router := mux.NewRouter()

	// A health checker for running in Docker or AppEngine.
	addRoute(router, "GET", "/_ah/health", http.HandlerFunc(checkHealth))

	// Stage2 scripts are always the first script fetched by a booting machine.
	// "stage2.ipxe" is the target for ROM-based iPXE clients.
	addRoute(router, "POST", "/v1/boot/{hostname}/stage2.ipxe",
		handler.Handler{env, handler.GenerateStage2IPXE})

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

func main() {
	if projectID == "" {
		log.Fatalf("Environment variable GCLOUD_PROJECT must specify a project ID for Datastore.")
	}
	if publicAddr == "" {
		log.Fatalf("Environment variable PUBLIC_ADDRESS must specify a public service name.")
	}

	// TODO(soltesz): support TLS natively for stand-alone mode. Though, this is not necessary for AppEngine.
	addr := fmt.Sprintf("%s:%s", bindAddress, bindPort)
	ctx := context.Background()
	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create new datastore client: %s", err)
	}
	env := &handler.Env{storage.NewDatastoreConfig(client), publicAddr}
	http.ListenAndServe(addr, newRouter(env))
}
