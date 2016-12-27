#!/bin/bash

set -x
set -e

gcloud auth activate-service-account --key-file /tmp/mlab-sandbox-appengine-deploy.json

pushd cmd/epoxy_boot_server
  gcloud --verbosity debug --project mlab-sandbox --quiet app deploy --promote app.yaml
popd
