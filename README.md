# ePoxy

A system for safe boot management over the Internet, based on iPXE.

## Building

To build the ePoxy boot server:

    go get github.com/m-lab/epoxy/cmd/epoxy_boot_server

## Deployment

The ePoxy server is designed to run from within a docker container. The M-Lab
deployment targets a stand-alone GCE VM. The cloudbuild.yaml configuration
embeds static zones for specific regional deployments for each GCP project.

Before deploying to a new Project complete the following steps in advance:

* Allocate static IP address and register DNS using `setup_epoxy_dns.sh`
* Allocate server certificte and key
* Create GCS bucket `gs://epoxy-${PROJECT}-private` and copy server certificate
  & key.

## Testing

### Testing Server

The datastore emulator depends on the [Google Cloud
SDK](https://cloud.google.com/sdk/downloads). After installing `gcloud`,
install the datastore emulator component:

    gcloud components install cloud-datastore-emulator

Next, start the datastore emulator:

    gcloud beta emulators datastore start

Look for the `DATASTORE_EMULATOR_HOST` reported on stdout. This environment
variable should be set for all subsequent commands.

Add a sample Host record to the Datastore emulator:

    TODO(soltesz): create command to add a minimal host record directly to DS.

Start the epoxy server:

    export DATASTORE_EMULATOR_HOST=< ... >
    export PUBLIC_ADDRESS=localhost:8080
    export GCLOUD_PROJECT="my-project"
    ./bin/epoxy_boot_server

The ePoxy server is now connected to the local datastore emulator, and can
serve client requests.

### Testing Client

After starting the datastore emuulator and a local epoxy boot server, you can
simulate a client request using `curl`.

    SERVER=localhost:8080
    curl --dump-header - --location -XPOST --data-binary "{}" \
        https://${SERVER}/v1/boot/mlab4.iad1t.measurement-lab.org/stage1.ipxe

If the host record is found in Datastore, then a stage1 boot script should be
returned. If the host record is not found, then:

    TODO(soltesz): handle 404 cases with a valid ipxe script.

If developing with the mlab-sandbox GCP, then verify that the deployment was
successful through travis and the AppEngine Cloud Console. Then set the SERVER
address for the boot-api service. For example, for mlab-sandbox, use:

    SERVER=boot-api-dot-mlab-sandbox.appspot.com
