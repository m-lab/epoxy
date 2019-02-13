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
// Managed machines progress through three boot stages:
//   stage1) local boot media like an iPXE ROM, or an immutable CD image
//   stage2) a minimal, linux-based network boot environment
//   stage3) the final system image.
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
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/m-lab/go/httpx"

	"cloud.google.com/go/datastore"
	"github.com/gorilla/mux"
	"github.com/m-lab/epoxy/handler"
	"github.com/m-lab/epoxy/metrics"
	"github.com/m-lab/epoxy/storage"
	"github.com/m-lab/go/rtx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/acme/autocert"
)

var (
	// Environment variables are preferred to flags for deployments in
	// AppEngine and Docker containers. Using environment variables is encouraged
	// for twelve-factor apps -- https://12factor.net/config

	// projectID must be set using the GCLOUD_PROJECT environment variable.
	projectID = os.Getenv("GCLOUD_PROJECT")

	// publicHostname must be set if *not* running in AppEngine. When running in
	// AppEngine publicHostname is set automatically from a combination of
	// GCLOUD_PROJECT and GAE_SERVICE environment variables. When not running in
	// AppEngine, or to override the default in AppEngine, the PUBLIC_HOSTNAME
	// environment variable must be set instead.
	publicHostname = os.Getenv("PUBLIC_HOSTNAME")

	// bindAddress may be set using the LISTEN environment variable. By default,
	// ePoxy listens on all available interfaces.
	bindAddress = os.Getenv("LISTEN")

	// bindPort may be set using the PORT environment variable.
	bindPort = "8080"

	// allowForwardedRequests controls how the ePoxy server evaluates and applies
	// the Host IP whitelist to incoming requests.
	// DEPRECATED.
	allowForwardedRequests = false

	// serverCert and serverKey are the filenames for the iPXE server certificate.
	serverCert = os.Getenv("IPXE_CERT_FILE")
	serverKey  = os.Getenv("IPXE_KEY_FILE")

	// enableLetsEncrypt enables allocation of let's encrypt certs. Should be
	// disabled for testing.
	enableLetsEncrypt = os.Getenv("ENABLE_LETSENCRYPT")

	// Create a unified context and a cancel method for main().
	ctx, cancelCtx = context.WithCancel(context.Background())
)

// init checks the environment for configuration values.
func init() {
	// Only use the automatic public address if PUBLIC_HOSTNAME is not already set.
	if service := os.Getenv("GAE_SERVICE"); service != "" && projectID != "" && publicHostname == "" {
		publicHostname = fmt.Sprintf("%s-dot-%s.appspot.com", service, projectID)
	}
	if port := os.Getenv("PORT"); port != "" {
		bindPort = port
	}
	if os.Getenv("ALLOW_FORWARDED_REQUESTS") == "true" {
		allowForwardedRequests = true
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

	///////////////////////////////////////////////////////////////////////////
	// Boot stage targets.
	//
	// Immediately after boot, a machine unconditionally requests a stage1 target.
	// After that, the machine should sequentially request the stage2, stage3, and
	// report targets in order. As each is requested, the session ID for the
	// previous is invalidated.

	// Stage1 scripts are always the first script fetched by a booting machine.
	// "stage1.ipxe" is the target for ROM-based iPXE clients.
	addRoute(router, "POST", "/v1/boot/{hostname}/stage1.ipxe",
		promhttp.InstrumentHandlerDuration(metrics.RequestDuration,
			http.HandlerFunc(env.GenerateStage1IPXE)))

	// "stage1.json" is the target for native ePoxy clients.
	addRoute(router, "POST", "/v1/boot/{hostname}/stage1.json",
		http.HandlerFunc(env.GenerateStage1JSON))

	// TODO: make the names stage2 and stage3 arbitrary when we need to support
	// the case where not every machine has the same stage2 or stage3.

	// TODO: consider placing stage sequence names under their own subpath, e.g.
	//   /v1/boot/{hostname}/{sessionID}/stage/{stage}"

	// Stage2, stage3, and report targets load after stage1 runs successfully. Stage2
	// and stage3 targets return an epoxy action. The report target returns no content.
	addRoute(router, "POST", "/v1/boot/{hostname}/{sessionID}/stage2",
		http.HandlerFunc(env.GenerateJSONConfig))
	addRoute(router, "POST", "/v1/boot/{hostname}/{sessionID}/stage3",
		http.HandlerFunc(env.GenerateJSONConfig))
	addRoute(router, "POST", "/v1/boot/{hostname}/{sessionID}/report",
		http.HandlerFunc(env.ReceiveReport))

	///////////////////////////////////////////////////////////////////////////
	// Extension targets.
	//
	// Extension operations may be requested at any time during boot. The session
	// is revoked after successful use. Extensions may return any content type
	// supported by the extension service.
	addRoute(router, "POST", "/v1/boot/{hostname}/{sessionID}/extension/{operation}",
		http.HandlerFunc(env.HandleExtension))

	return router
}

// checkHealth reports whether the server is healthy. checkHealth will
// typically be registered as the http.Handler for the path "/_ah/health" when
// running in Docker or AppEngine.
func checkHealth(rw http.ResponseWriter, req *http.Request) {
	fmt.Fprint(rw, "ok")
}

func setupMetricsHandler(dsCfg *storage.DatastoreConfig) *http.ServeMux {
	// Note: we use custom collectors to read directly from datastore rather than
	// instrumenting http handlers because we want to guarantee that metrics are
	// always available, even after an appengine server restart. These metrics will
	// be critical for defining alerts on boot failures.
	prometheus.Register(metrics.NewCollector("epoxy_last_boot", dsCfg))
	prometheus.Register(metrics.NewCollector("epoxy_last_success", dsCfg))
	// Define a custom serve mux for prometheus to listen on a separate port.
	// We listen on a separate port so we can forward this port on the host VM.
	// We cannot forward port 8080 because it is used by AppEngine.
	mux := http.NewServeMux()
	// Assign the default prometheus handler to the standard exporter path.
	mux.Handle("/metrics", promhttp.Handler())
	return mux
}

func setupPXEServer(addr string, r *mux.Router) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: r,
	}
}

func setupLetsEncryptServer(addr string, r *mux.Router, hostname string) *http.Server {
	// We will listen on standard TLS port using LetsEncrypt certificates.
	m := &autocert.Manager{
		// Certificates are cached to a local directory.
		Cache: autocert.DirCache("autocert.cache"),
		// The "Let's Encrypt Terms of Service" are accepted automatically.
		Prompt: autocert.AcceptTOS,
		// The ePoxy server will only accept TLS host requests from given hostname.
		HostPolicy: autocert.HostWhitelist(hostname),
	}
	// Server with custom TLS config.
	return &http.Server{
		Addr:      addr,
		Handler:   r,
		TLSConfig: m.TLSConfig(),
	}
}

func startAppEngineServerAsync(addr string, router *mux.Router) {
	ipxeServer := setupPXEServer(addr, router)
	httpx.ListenAndServeAsync(ipxeServer)
}

func startTLSServerAsync(addr string, router *mux.Router, hostname string) {
	tlsServer := setupLetsEncryptServer(addr, router, hostname)
	// Certificates are already configured in the server.TLSConfig.
	httpx.ListenAndServeTLSAsync(tlsServer, "", "")

	// Because we're running LetsEncrypt certificates on the standard TLS port,
	// run the iPXE server on a higher port.
	ipxeServer := setupPXEServer(addr+"0", router)
	if serverCert == "" || serverKey == "" {
		log.Fatalln("WARNING: IPXE_CERT_FILE and IPXE_KEY_FILE were not specified.")
	}
	httpx.ListenAndServeTLSAsync(ipxeServer, serverCert, serverKey)
}

func main() {
	defer cancelCtx()

	if projectID == "" {
		log.Fatalf("Environment variable GCLOUD_PROJECT must specify a project ID for Datastore.")
	}
	if publicHostname == "" {
		log.Fatalf("Environment variable PUBLIC_HOSTNAME must specify a public service name.")
	}

	tlsAddr := fmt.Sprintf("%s:%s", bindAddress, bindPort)
	pxeAddr := tlsAddr
	// addr := fmt.Sprintf("%s:%s", bindAddress, bindPort)
	client, err := datastore.NewClient(ctx, projectID)
	rtx.Must(err, "Failed to create new datastore client")

	dsCfg := storage.NewDatastoreConfig(client)
	env := &handler.Env{
		Config:                 dsCfg,
		ServerAddr:             publicHostname,
		AllowForwardedRequests: allowForwardedRequests,
	}

	go func() {
		mux := setupMetricsHandler(dsCfg)
		log.Fatal(http.ListenAndServe(":9090", mux))
	}()

	router := newRouter(env)
	if service := os.Getenv("GAE_SERVICE"); service != "" {
		// Assume we are running in AppEngine.
		startAppEngineServerAsync(pxeAddr, router)
	} else {
		startTLSServerAsync(tlsAddr, router, publicHostname)
	}

	<-ctx.Done()

}
