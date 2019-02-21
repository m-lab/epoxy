#!/bin/bash

set -ex

ZONE_mlab_sandbox=us-east1-d
IP_mlab_sandbox=35.190.184.60

ip_ref=IP_${PROJECT//-/_}
zone_ref=ZONE_${PROJECT//-/_}

ZONE=${!zone_ref}
IP=${!ip_ref}

# Lookup the instance (if any) currently using the static IP address for ePoxy.
gce_url=$( gcloud compute addresses describe --project "${PROJECT}" \
		     --format "value(users)" --region "${ZONE%-*}" epoxy-boot-api )
CURRENT_INSTANCE=${gce_url##*/}
UPDATED_INSTANCE="epoxy-boot-api-$( date +%Y%m%dt%H%M%S )"

CERTDIR=/home/epoxy
# Create startup script to pass to create instance. Script will run as root.
cat <<EOF > startup.sh
#!/bin/bash
set -x
mkdir ${CERTDIR}
# Copy certificates from GCS.
until docker run --tty --volume "${CERTDIR}:${CERTDIR}" \
    gcr.io/cloud-builders/gsutil \
	cp gs://epoxy-${PROJECT}-private/server-certs.pem \
	   gs://epoxy-${PROJECT}-private/server-key.pem \
	   ${CERTDIR} ; do
  sleep 5
done
EOF

cat <<EOF > config.env
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
    --container-image "soltesz/epoxy_boot_server" \
    --container-mount-host-path host-path=/home/epoxy,mount-path=/certs \
    --container-env-file config.env

sleep 20
TEMP_IP=$( gcloud compute instances describe \
   --project "${PROJECT}" --zone "${ZONE}" \
   --format 'value(networkInterfaces[].accessConfigs[0].natIP)' \
   ${UPDATED_INSTANCE} )

# Run a basic diagnostic test.
while ! curl --insecure --dump-header - https://${TEMP_IP}:4430/_ah/health ; do
   sleep 5
done

# Remove public IP from updated instance so we can assign the (now available)
# static IP.
gcloud compute instances delete-access-config --zone "${ZONE}" \
   --project "${PROJECT}" \
   --access-config-name "external-nat" "${UPDATED_INSTANCE}"

if [[ -n "${CURRENT_INSTANCE}" ]] ; then
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
