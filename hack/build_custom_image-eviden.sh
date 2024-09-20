#!/bin/bash
# This script creates a custom container image for both, Barbican API and Barbican Worker,
# including the HSM vendor's client software.
# Vendor:  Eviden

if [ "$#" -ne 5 ]; then
    echo "Usage: $0 <registry_host> <namespace> <barbican-api_image_tag> <barbican-worker_image_tag> <eviden_iso_file>"
    exit 1
fi

REGISTRY_HOST=$1
NAMESPACE=$2
API_IMAGE_TAG=$3
WORKER_IMAGE_TAG=$4
EVIDEN_ISO_FILE=$5
TEMP_ISO_DIR=iso_eviden
USERNAME=replace_with_your_registry_username

echo
echo "You need to be logged into your registry for this script to work."
echo "If you're not logged in, stop this script now and log in with 'podman login'."

echo
echo "Downloading Barbican API image..."
podman pull $REGISTRY_HOST/$NAMESPACE/openstack-barbican-api:$API_IMAGE_TAG
echo
echo "Downloading Barbican Worker image..."
podman pull $REGISTRY_HOST/$NAMESPACE/openstack-barbican-worker:$WORKER_IMAGE_TAG

echo
echo "Locally mounting ISO client file..."
mkdir $TEMP_ISO_DIR
sudo mount -o loop $EVIDEN_ISO_FILE $TEMP_ISO_DIR
if [ "$?" -ne 0 ]; then
    echo "Unable to locally mount the HSM client file. Exiting."
    exit 2
fi

echo
echo "Creating Dockerfile for barbican-api..."
cat <<EOF > Dockerfile.barbican-api
FROM $REGISTRY_HOST/$NAMESPACE/openstack-barbican-api:$API_IMAGE_TAG

USER root
RUN mkdir /tmp/iso
COPY $TEMP_ISO_DIR/ /tmp/iso/

RUN cd /tmp/iso/Linux; { echo "e"; echo "n"; echo; } | bash install.sh
RUN rm -rf /tmp/iso
EOF

echo
echo "Creating Dockerfile for barbican-worker..."
cat <<EOF > Dockerfile.barbican-worker
FROM $REGISTRY_HOST/$NAMESPACE/openstack-barbican-worker:$API_IMAGE_TAG

USER root
RUN mkdir /tmp/iso
COPY $TEMP_ISO_DIR/ /tmp/iso/

RUN cd /tmp/iso/Linux; { echo "e"; echo "n"; echo; } | bash install.sh
RUN rm -rf /tmp/iso
EOF

echo
echo "Building new container images..."
buildah bud -t barbican-api-custom:$API_IMAGE_TAG -f Dockerfile.barbican-api
buildah bud -t barbican-worker-custom:$WORKER_IMAGE_TAG -f Dockerfile.barbican-worker

echo "Pushing new images to the registry..."
# Replace the registry URL with the appropriate one for your environment
REGISTRY_URL=$(REGISTRY_HOST)/$(USERNAME)

podman tag barbican-api-custom:$API_IMAGE_TAG $REGISTRY_URL/barbican-api-custom:$API_IMAGE_TAG
podman push $REGISTRY_URL/barbican-api-custom:$API_IMAGE_TAG

podman tag barbican-worker-custom:$WORKER_IMAGE_TAG $REGISTRY_URL/barbican-worker-custom:$WORKER_IMAGE_TAG
podman push $REGISTRY_URL/barbican-worker-custom:$WORKER_IMAGE_TAG

echo
echo "Unmounting ISO client file..."
sudo umount $TEMP_ISO_DIR
if [ "$?" -ne 0 ]; then
    echo "Unable to unmount the HSM client file. Do it manually."
    exit 3
fi
rmdir $TEMP_ISO_DIR
echo "Done."
