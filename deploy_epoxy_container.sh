#!/bin/bash
#
# deploy_epoxy_container.sh creates a GCE VM, which runs a startup script, and
# starts a container image for the epoxy_boot_server. Because the epoxy boot api
# uses a static IP, that IP is unassigned from the current VM and reassigned to
# the new VM, once the new VM appears to be working.
#
# In addition to VM creation, deploy_epoxy_container.sh also sets up the
# pre-conditions for the ePoxy server GCP networking by creating an "epoxy"
# subnet of the provided NETWORK, allocating a static IP for the server and
# assigning it to a DNS record.
#
# deploy_epoxy_container.sh depends on three environment variables for correct
# operation.
#
#  PROJECT - specifies the GCP project name to create the VM, e.g. mlab-sandbox.
#  CONTAINER - specifies the docker container image URL, e.g.
#                gcr.io/mlab-sandbox/epoxy_boot_server
#  ZONE_<project> - specifies the GCP VM zone, e.g. ZONE_mlab_sandbox=us-east1-c
#  NETWORK - specifies the GCP network name shared with the k8s platform cluster
#
# Example usage:
#
#   PROJECT=mlab-sandbox \
#   CONTAINER=gcr.io/mlab-sandbox/epoxy_boot_server:$BUILD_ID \
#   ZONE_mlab_sandbox=us-east1-c \
#   NETWORK=mlab-platform-network \
#     ./deploy_epoxy_container.sh

set -uex

zone_ref=ZONE_${PROJECT//-/_}
ZONE=${!zone_ref}
REGION="${ZONE%-*}"

EPOXY_SUBNET="epoxy"
ARGS=("--project=${PROJECT}" "--quiet")

if [[ -z "${PROJECT}" || -z "${CONTAINER}" || -z "${ZONE}" ]]; then
  echo "ERROR: PROJECT, CONTAINER, and ZONE must be defined in environment."
  echo "ERROR: Current values are:"
  echo "  PROJECT='${PROJECT}'"
  echo "  CONTAINER='${CONTAINER}'"
  echo "  ZONE='${ZONE}'"
  exit 1
fi

if [[ -z "${NETWORK}" ]]; then
  echo "ERROR: NETWORK= shared with platform-cluster must be defined."
  exit 1
fi

# Find the lowest network number available for a new epoxy subnet.
function find_lowest_network_number() {
  local current_sequence=$( mktemp )
  local natural_sequence=$( mktemp )
  local available=$( mktemp )

  # List current network subnets, and extract the second octet from each.
  gcloud compute networks subnets list \
    --network "${NETWORK}" --format "value(ipCidrRange)" "${ARGS[@]}" \
    | cut -d. -f2 | sort -n > "${current_sequence}"

  # Generate a natural sequence from 0 to 255.
  seq 0 255 > "${natural_sequence}"

  # Find values present in $natural_sequence but missing from $current_sequence.
  # -1 = suppress lines unique to file 1
  # -3 = suppress lines that appear in both files
  # As a result, only report lines that are unique to "${natural_sequence}".
  comm -1 -3 --nocheck-order \
    "${current_sequence}" "${natural_sequence}" > "${available}"

  # "Return" the first $available value: the lowest available network number.
  head -n 1 "${available}"

  # Clean up temporary files.
  rm -f "${current_sequence}" "${natural_sequence}" "${available}"
}


###############################################################################
## Setup ePoxy subnet if not found in current region.
###############################################################################
# Try to find an epoxy subnet in the current region.
EPOXY_SUBNET_IN_REGION=$(gcloud compute networks subnets list \
  --network "${NETWORK}" \
  --filter "name=${EPOXY_SUBNET} AND region:${REGION}" \
  --format "value(name)" \
  "${ARGS[@]}" || : )
if [[ -z "${EPOXY_SUBNET_IN_REGION}" ]]; then
  # If it doesn't exist, then create it with the first available network.
  N=$( find_lowest_network_number )
  gcloud compute networks subnets create "${EPOXY_SUBNET}" \
    --network "${NETWORK}" \
    --range "10.${N}.0.0/16" \
    --region "${REGION}" \
    "${ARGS[@]}"
fi

###############################################################################
## Setup ePoxy server static IP & DNS.
###############################################################################
IP=$(
  gcloud compute addresses describe epoxy-boot-api \
    --format "value(address)" --region "${REGION}" "${ARGS[@]}" || : )
if [[ -z "${IP}" ]] ; then
  gcloud compute addresses create epoxy-boot-api \
    --region "${REGION}" "${ARGS[@]}"
  IP=$(
    gcloud compute addresses describe epoxy-boot-api \
      --format "value(address)" --region "${REGION}" "${ARGS[@]}" )
fi
if [[ -z "${IP}" ]]; then
  echo "ERROR: Failed to find or allocate static IP in region ${REGION}"
  exit 1
fi

CURRENT_IP=$(
  gcloud dns record-sets list --zone "${PROJECT}-measurementlab-net" \
    --name "epoxy-boot-api.${PROJECT}.measurementlab.net." \
    --format "value(rrdatas[0])" "${ARGS[@]}" )
if [[ "${CURRENT_IP}" != "${IP}" ]] ; then
  # Add the record, deleting the existing one first.
  gcloud dns record-sets transaction start \
    --zone "${PROJECT}-measurementlab-net" \
    "${ARGS[@]}"
  # Allow remove to fail when CURRENT_IP is empty.
  gcloud dns record-sets transaction remove \
    --zone "${PROJECT}-measurementlab-net" \
    --name "epoxy-boot-api.${PROJECT}.measurementlab.net." \
    --type A \
    --ttl 300 \
    "${CURRENT_IP}" \
    "${ARGS[@]}" || :
  gcloud dns record-sets transaction add \
    --zone "${PROJECT}-measurementlab-net" \
    --name "epoxy-boot-api.${PROJECT}.measurementlab.net." \
    --type A \
    --ttl 300 \
    "${IP}" \
    "${ARGS[@]}"
  gcloud dns record-sets transaction execute \
    --zone "${PROJECT}-measurementlab-net" \
    "${ARGS[@]}"
fi

###############################################################################
## Deploy ePoxy server.
###############################################################################
# Lookup the instance (if any) currently using the static IP address for ePoxy.
gce_url=$(gcloud compute addresses describe --project "${PROJECT}" \
  --format "value(users)" --region "${REGION}" epoxy-boot-api)
CURRENT_INSTANCE=${gce_url##*/}
UPDATED_INSTANCE="epoxy-boot-api-$(date +%Y%m%dt%H%M%S)"

CERTDIR=/home/epoxy
# Create startup script to pass to create instance. Script will run as root.
cat <<EOF >startup.sh
#!/bin/bash
set -x
mkdir "${CERTDIR}"
# Build the GCS fuse user space tools.
# Retry because docker fails to contact gcr.io sometimes.
until docker run --rm --tty --volume /var/lib/toolbox:/tmp/go/bin \
  --env "GOPATH=/tmp/go" \
  amd64/golang:1.11.5 /bin/bash -c \
   "go get -u github.com/googlecloudplatform/gcsfuse &&
    apt-get update --quiet=2 &&
    apt-get install --yes fuse &&
    cp /bin/fusermount /tmp/go/bin" ; do
  sleep 5
done

mkdir ${CERTDIR}/bucket
export PATH=\$PATH:/var/lib/toolbox
/var/lib/toolbox/gcsfuse --implicit-dirs -o rw,allow_other \
  epoxy-${PROJECT}-private ${CERTDIR}/bucket
EOF

cat <<EOF >config.env
IPXE_CERT_FILE=/certs/server-certs.pem
IPXE_KEY_FILE=/certs/server-key.pem
PUBLIC_HOSTNAME=epoxy-boot-api.${PROJECT}.measurementlab.net
STORAGE_PREFIX_URL=https://storage.googleapis.com/epoxy-${PROJECT}
GCLOUD_PROJECT=${PROJECT}
EOF

CURRENT_FIREWALL=$(
  gcloud compute firewall-rules list \
    --filter "name=allow-epoxy-ports" --format "value(name)" "${ARGS[@]}" )
if [[ -z "${CURRENT_FIREWALL}" ]]; then
  # Create a new firewall to open access for all epoxy boot server ports.
  gcloud compute firewall-rules create "allow-epoxy-ports" \
    --project "${PROJECT}" \
    --network "mlab-platform-network" \
    --action "allow" \
    --rules "tcp:443,tcp:4430,tcp:9000" \
    --target-tags "allow-epoxy-ports" \
    --source-ranges "0.0.0.0/0" || :
fi

# Create new VM with ephemeral public IP.
gcloud compute instances create-with-container "${UPDATED_INSTANCE}" \
  --project "${PROJECT}" \
  --zone "${ZONE}" \
  --tags allow-epoxy-ports \
  --scopes default,datastore,storage-full \
  --metadata-from-file "startup-script=startup.sh" \
  --network-interface network=mlab-platform-network,subnet=epoxy \
  --container-image "${CONTAINER}" \
  --container-mount-host-path host-path=${CERTDIR}/bucket,mount-path=/certs \
  --container-env-file config.env

sleep 20
TEMP_IP=$(
  gcloud compute instances describe ${UPDATED_INSTANCE} \
    --format 'value(networkInterfaces[].accessConfigs[0].natIP)' \
    --zone "${ZONE}" "${ARGS[@]}" )

# Run a basic diagnostic test.
while ! curl --insecure --dump-header - https://${TEMP_IP}:4430/_ah/health; do
  sleep 5
done

# Remove public IP from updated instance so we can assign the (now available)
# static IP.
gcloud compute instances delete-access-config --zone "${ZONE}" \
  "${ARGS[@]}" \
  --access-config-name "external-nat" "${UPDATED_INSTANCE}"

if [[ -n "${CURRENT_INSTANCE}" ]]; then
  # Remove public IP from current instance so we can assign it to the new one.
  gcloud compute instances delete-access-config --zone "${ZONE}" \
    "${ARGS[@]}" \
    --access-config-name "external-nat" "${CURRENT_INSTANCE}"
fi

# Assign the static IP to the updated instance.
gcloud compute instances add-access-config --zone "${ZONE}" \
  "${ARGS[@]}" \
  --access-config-name "external-nat" --address "$IP" \
  "${UPDATED_INSTANCE}"

echo "Success!"
