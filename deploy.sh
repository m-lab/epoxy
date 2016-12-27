#!/bin/bash

set -x
set -e

env
pwd

gcloud auth activate-service-account --key-file /tmp/mlab-sandbox-appengine-deploy.json

pushd $TRAVIS_BUILD_DIR/cmd/epoxy_boot_server
  gcloud --verbosity debug --project mlab-sandbox --quiet app deploy --promote app.yaml
popd

exit 0
