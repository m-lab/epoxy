#!/bin/bash

set -x
set -e

PROJECT=${1:?Please provide the project}
KEYFILE=${2:?Please provide the service account key file}
BASEDIR=${3:?Please provide the base directory containing app.yaml}

source "${HOME}/google-cloud-sdk/path.bash.inc"

gcloud auth activate-service-account --key-file "${KEYFILE}"

pushd "${BASEDIR}"
  # --quiet suppresses prompts for user input.
  gcloud --quiet --verbosity debug --project "${PROJECT}" \
      app deploy --promote app.yaml
popd

exit 0
