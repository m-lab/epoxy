#!/bin/bash
#
# deploy_epoxy_container.sh creates a GCE VM, which runs a startup script, and
# starts a container image for the epoxy_boot_server. Because the epoxy boot api
# uses a static IP, that IP is unassigned from the current VM and reassigned to
# the new VM, once the new VM appears to be working.
#
# deploy_epoxy_container.sh depends on three environment variables for correct
# operation.
#
#  PROJECT - specifies the GCP project name to create the VM, e.g. mlab-sandbox.
#  CONTAINER - specifies the docker container image URL, e.g.
#                gcr.io/mlab-sandbox/epoxy_boot_server
#  ZONE_<project> - specifies the GCP VM zone, e.g. ZONE_mlab_sandbox=us-east1-c
#
# Example usage:
#
#   PROJECT=mlab-sandbox \
#   CONTAINER=gcr.io/mlab-sandbox/epoxy_boot_server:$BUILD_ID \
#   ZONE_mlab_sandbox=us-east1-c \
#     ./deploy_epoxy_container.sh

set -ex

zone_ref=ZONE_${PROJECT//-/_}
ZONE=${!zone_ref}

if [[ -z "${PROJECT}" || -z "${CONTAINER}" || -z "${ZONE}" ]]; then
  echo "ERROR: Failed to lookup GCP ZONE ('$ZONE') for project '$PROJECT'"
  exit 1
fi

# Lookup address.
IP=$(gcloud compute addresses describe --project "${PROJECT}" \
  --format "value(address)" --region "${ZONE%-*}" epoxy-boot-api)
if [[ -z "${IP}" ]]; then
  echo "ERROR: Failed to find static IP in region ${ZONE%-*}"
  echo "ERROR: Run the m-lab/epoxy/setup_epoxy_dns.sh to allocate one."
  exit 1
fi
# Lookup the instance (if any) currently using the static IP address for ePoxy.
gce_url=$(gcloud compute addresses describe --project "${PROJECT}" \
  --format "value(users)" --region "${ZONE%-*}" epoxy-boot-api)
CURRENT_INSTANCE=${gce_url##*/}
UPDATED_INSTANCE="epoxy-boot-api-$(date +%Y%m%dt%H%M%S)"

CERTDIR=/home/epoxy
# Create startup script to pass to create instance. Script will run as root.
cat <<EOF >startup.sh
#!/bin/bash
set -x
mkdir "${CERTDIR}"
# Copy certificates from GCS.
# Retry because docker fails to contact gcr.io sometimes.
until docker run --tty --volume "${CERTDIR}:${CERTDIR}" \
  gcr.io/cloud-builders/gsutil \
  cp gs://epoxy-${PROJECT}-private/server-certs.pem \
      gs://epoxy-${PROJECT}-private/server-key.pem \
      "${CERTDIR}"; do
  sleep 5
done
EOF

cat <<EOF >config.env
IPXE_CERT_FILE=/certs/server-certs.pem
IPXE_KEY_FILE=/certs/server-key.pem
PUBLIC_HOSTNAME=epoxy-boot-api.${PROJECT}.measurementlab.net
STORAGE_PREFIX_URL=https://storage.googleapis.com/epoxy-${PROJECT}
GCLOUD_PROJECT=${PROJECT}
EOF

# Create new VM without public IP.
gcloud compute instances create-with-container "${UPDATED_INSTANCE}" \
  --project "${PROJECT}" \
  --zone "${ZONE}" \
  --tags allow-epoxy-ports \
  --scopes default,datastore \
  --metadata-from-file "startup-script=startup.sh" \
  --network-interface network=mlab-platform-network,subnet=epoxy \
  --container-image "${CONTAINER}" \
  --container-mount-host-path host-path=/home/epoxy,mount-path=/certs \
  --container-env-file config.env

sleep 20
TEMP_IP=$(gcloud compute instances describe \
  --project "${PROJECT}" --zone "${ZONE}" \
  --format 'value(networkInterfaces[].accessConfigs[0].natIP)' \
  ${UPDATED_INSTANCE})

# Run a basic diagnostic test.
while ! curl --insecure --dump-header - https://${TEMP_IP}:4430/_ah/health; do
  sleep 5
done

# Remove public IP from updated instance so we can assign the (now available)
# static IP.
gcloud compute instances delete-access-config --zone "${ZONE}" \
  --project "${PROJECT}" \
  --access-config-name "external-nat" "${UPDATED_INSTANCE}"

if [[ -n "${CURRENT_INSTANCE}" ]]; then
  # Remove public IP from current instance so we can assign it to the new one.
  gcloud compute instances delete-access-config --zone "${ZONE}" \
    --project "${PROJECT}" \
    --access-config-name "external-nat" "${CURRENT_INSTANCE}"
fi

# Assign the static IP to the updated instance.
gcloud compute instances add-access-config --zone "${ZONE}" \
  --project "${PROJECT}" \
  --access-config-name "external-nat" --address "$IP" \
  "${UPDATED_INSTANCE}"
