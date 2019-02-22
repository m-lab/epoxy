#!/bin/bash
#
# setup_epoxy_dns.sh looks up the current epoxy-boot-api static IP. If one is
# not found, a new one is created. Then, setup_epoxy_dns.sh looks up the
# current DNS record for epoxy-boot-api.<project>.measurementlab.net and
# updates it if needed.
#
# setup_epoxy_dns.sh should be safe to run multiple times.
#
# EXAMPLE USAGE:
#   PROJECT=mlab-sandbox ZONE=us-east1-c ./setup_epoxy_dns.sh

set -xe

IP=$( gcloud compute addresses describe --project "${PROJECT}" \
        --format "value(address)" --region "${ZONE%-*}" epoxy-boot-api || : )
if [[ -z "${IP}" ]] ; then
    gcloud compute addresses create epoxy-boot-api \
        --project "${PROJECT}" \
        --region "${ZONE%-*}"
    IP=$( gcloud compute addresses describe --project "${PROJECT}" \
            --format "value(address)" --region "${ZONE%-*}" epoxy-boot-api )
fi

CURRENT_IP=$(
    gcloud dns record-sets list --zone "${PROJECT}-measurementlab-net" \
       --name "epoxy-boot-api.${PROJECT}.measurementlab.net." \
       --format "value(rrdatas[0])" --project "${PROJECT}" )
if [[ "${CURRENT_IP}" != "${IP}" ]] ; then

    # Add the record, deleting the existing one first.
    gcloud dns record-sets transaction start \
        --zone "${PROJECT}-measurementlab-net" \
        --project "${PROJECT}"
	# Allow remove to fail when CURRENT_IP is empty.
    gcloud dns record-sets transaction remove \
        --zone "${PROJECT}-measurementlab-net" \
        --name "epoxy-boot-api.${PROJECT}.measurementlab.net." \
        --type A \
        --ttl 300 \
        "${CURRENT_IP}" \
        --project "${PROJECT}" || :
    gcloud dns record-sets transaction add \
        --zone "${PROJECT}-measurementlab-net" \
        --name "epoxy-boot-api.${PROJECT}.measurementlab.net." \
        --type A \
        --ttl 300 \
        "${IP}" \
        --project "${PROJECT}"
    gcloud dns record-sets transaction execute \
        --zone "${PROJECT}-measurementlab-net" \
        --project "${PROJECT}"

fi
