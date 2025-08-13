#!/bin/bash

set -e

DEFAULT_ZONE=us-central1-a
DEFAULT_ARCH=amd64
DEFAULT_GCLOUD_COMMAND=gcloud
usage() {
    echo "Usage: $0 -p <PROJECT> [-z <ZONE>] [-a <ARCH>] [-c <GCLOUD_COMMAND>]"
    echo "  -p, --project        : GCP project name, must be specified"
    echo "  -z, --zone           : GCE zone (default: $DEFAULT_ZONE)"
    echo "  -a, --arch           : Arch of GCE image (default: $DEFAULT_ARCH)"
    echo "  -c, --gcloud_command : Gcloud command (default: $DEFAULT_GCLOUD_COMMAND)"
}

PROJECT=
ZONE=$DEFAULT_ZONE
ARCH=$DEFAULT_ARCH
GCLOUD_COMMAND=$DEFAULT_GCLOUD_COMMAND
parse_args() {
    while [ $# -gt 0 ]; do
        case "$1" in
            -h|--help)
                usage
                exit 0
                ;;
            -p|--project)
                PROJECT="$2"
                shift 2
                ;;
            -z|--zone)
                ZONE="$2"
                shift 2
                ;;
            -a|--arch)
                ARCH="$2"
                shift 2
                ;;
            -c|--gcloud_command)
                GCLOUD_COMMAND="$2"
                shift 2
                ;;
            *)
                echo "Invalid command" >&2
                usage
                exit 1
                ;;
        esac
    done
    if [ -z "$PROJECT" ]; then
        echo "Missing project"
        exit 1
    fi
    if [ -z "$ZONE" ]; then
        echo "Missing zone"
        exit 1
    fi
    case "$ARCH" in
        "amd64"|"x86_64")
            ARCH="amd64"
            ;;
        *)
            echo "Unsupported arch $ARCH"
            exit 1
            ;;
    esac
    if [ -z "$GCLOUD_COMMAND" ]; then
        echo "Missing gcloud command"
        exit 1
    fi
    echo "Project : $PROJECT"
    echo "Zone    : $ZONE"
    echo "Arch    : $ARCH"
}

RANDOM_SUFFIX=$(cat /dev/urandom | tr -dc 'a-z0-9' | head -c 8)
IMAGE_BUILDER_INSTANCE=cf-image-builder-$RANDOM_SUFFIX
IMAGE_NAME=cf-cloud-orchestrator-$RANDOM_SUFFIX

create_image_builder_instance() {
    $GCLOUD_COMMAND --project $PROJECT \
        compute instances create $IMAGE_BUILDER_INSTANCE \
        --image-project=debian-cloud \
        --image-family=debian-12 \
        --machine-type=n1-standard-1 \
        --zone=$ZONE
}

delete_image_builder_instance() {
    $GCLOUD_COMMAND --project $PROJECT \
        compute instances delete $IMAGE_BUILDER_INSTANCE \
        --quiet \
        --zone=$ZONE
}

exec_command_on_image_builder_instance() {
    $GCLOUD_COMMAND --project $PROJECT \
        compute ssh $IMAGE_BUILDER_INSTANCE \
        --zone=$ZONE \
        --command="$1"
}

prepare_image_builder_instance() {
    case "${DEFAULT_ZONE%%-*}" in
            "asia")
                DOWNLOAD_LOCATION=asia
                ;;
            "europe")
                DOWNLOAD_LOCATION=europe
                ;;
            *)
                DOWNLOAD_LOCATION=us
                ;;
    esac
    until exec_command_on_image_builder_instance "echo 'GCE instance is now ready'"
    do
        echo 'GCE instance is not ready yet, waiting...'
        sleep 5
    done
    exec_command_on_image_builder_instance "
        sudo install -m 0755 -d /etc/apt/keyrings &&
        sudo curl -fsSL https://download.docker.com/linux/debian/gpg \
            -o /etc/apt/keyrings/docker.asc &&
        sudo chmod a+r /etc/apt/keyrings/docker.asc &&
        echo \"deb [arch=$ARCH signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable\" |\
            sudo tee /etc/apt/sources.list.d/docker.list > /dev/null &&
        echo \"deb http://deb.debian.org/debian bookworm-backports main\" |\
            sudo tee /etc/apt/sources.list > /dev/null &&
        sudo apt update &&
        sudo apt install -y linux-base -t bookworm-backports &&
        sudo apt install -y linux-image-cloud-$ARCH -t bookworm-backports &&
        sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin &&
        sudo docker pull \
            $DOWNLOAD_LOCATION-docker.pkg.dev/android-cuttlefish-artifacts/cuttlefish-orchestration/cuttlefish-cloud-orchestrator &&
        sudo docker pull \
            $DOWNLOAD_LOCATION-docker.pkg.dev/android-cuttlefish-artifacts/cuttlefish-orchestration/cuttlefish-orchestration &&
        sudo mkdir -p /etc/cloud_orchestrator &&
        sudo wget -O /etc/cloud_orchestrator/conf.toml \
            https://artifactregistry.googleapis.com/download/v1/projects/android-cuttlefish-artifacts/locations/$DOWNLOAD_LOCATION/repositories/cloud-orchestrator-config/files/on-premise-single-server:main:conf.toml:download?alt=media &&
        sudo docker run --restart unless-stopped -d -p 8080:8080 -e CONFIG_FILE="/conf.toml" \
            -v /etc/cloud_orchestrator/conf.toml:/conf.toml \
            -v /var/run/docker.sock:/var/run/docker.sock \
            -t $DOWNLOAD_LOCATION-docker.pkg.dev/android-cuttlefish-artifacts/cuttlefish-orchestration/cuttlefish-cloud-orchestrator:latest
    "
}

stop_image_builder_instance() {
    $GCLOUD_COMMAND --project $PROJECT \
        compute instances stop $IMAGE_BUILDER_INSTANCE \
        --zone=$ZONE
}

create_image() {
    $GCLOUD_COMMAND --project $PROJECT \
        compute images create $IMAGE_NAME \
        --family=cf-cloud-orchestrator-$ARCH \
        --source-disk=$IMAGE_BUILDER_INSTANCE \
        --source-disk-zone=$ZONE \
        --storage-location=us-central1 \
        --guest-os-features=IDPF
}

parse_args $@
create_image_builder_instance
trap delete_image_builder_instance EXIT ERR
prepare_image_builder_instance
stop_image_builder_instance
create_image
