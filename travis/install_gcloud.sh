#!/bin/bash
#
# Installs the Google Cloud SDK.

set -e
set -x

if [[ ! -d "${HOME}/google-cloud-sdk/bin" ]]; then
  rm -rf "${HOME}/google-cloud-sdk"

  export CLOUDSDK_CORE_DISABLE_PROMPTS=1
  curl https://sdk.cloud.google.com | bash

fi

# Verify installation succeeded.
source "${HOME}/google-cloud-sdk/path.bash.inc"
gcloud version
