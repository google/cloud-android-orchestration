#!/bin/bash

set -e

if [ ! -f $(which cvdr) ]; then
    # Install cvdr from JFrog repository
    sudo apt install -y wget
    wget -qO- https://artifacts.codelinaro.org/artifactory/linaro-372-googlelt-gigabyte-ampere-cuttlefish-installer/gigabyte-ampere-cuttlefish-installer/latest/debian/linaro-glt-gig-archive-bookworm.asc | sudo tee /etc/apt/trusted.gpg.d/linaro-glt-gig-archive-bookworm.asc
    echo "deb https://artifacts.codelinaro.org/linaro-372-googlelt-gigabyte-ampere-cuttlefish-installer/gigabyte-ampere-cuttlefish-installer/latest/debian bookworm main" | sudo tee /etc/apt/sources.list.d/linaro-glt-gig-archive-bookworm.list
    sudo apt update && sudo apt install -y cuttlefish-cvdremote
fi

NUM_HOSTS=1
while true; do
    case "$1" in
        -n|--num_hosts)
            NUM_HOSTS="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [--num_hosts <num_hosts>] -- [<subcommands of cvdr create>]"
            exit 0
            ;;
        --)
            shift
            break
            ;;
        *)
            echo "Invalid command" >&2
            exit 1
            ;;
    esac
done

echo "Creation of the first host started"
./cvdr create $@
echo "Created first host"

echo "Creation of the rest hosts started"
for i in $(seq 2 $NUM_HOSTS); do
    ./cvdr create $@ > /dev/null &
done
wait
echo "Created all hosts"
