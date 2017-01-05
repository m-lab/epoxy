#!/bin/bash
#
# Performs an AppEngine deployment using service account credentials.

set -x
set -e

PROJECT=${1:?Please provide the GCP project id}
KEYFILE=${2:?Please provide the service account key file}
BASEDIR=${3:?Please provide the base directory containing app.yaml}

# Add gcloud to PATH.
source "${HOME}/google-cloud-sdk/path.bash.inc"

# All operations are performed as the service account named in KEYFILE.
# For all options see:
# https://cloud.google.com/sdk/gcloud/reference/auth/activate-service-account
gcloud auth activate-service-account --key-file "${KEYFILE}"

# For all options see:
# https://cloud.google.com/sdk/gcloud/reference/config/set
gcloud config set core/project "${PROJECT}"
gcloud config set core/disable_prompts true
gcloud config set core/verbosity debug

# Make build artifacts available to docker build.
pushd "${BASEDIR}"
  # Automatically promote the new version to "serving".
  # For all options see:
  # https://cloud.google.com/sdk/gcloud/reference/app/deploy
  gcloud app deploy --promote app.yaml
popd

exit 0
