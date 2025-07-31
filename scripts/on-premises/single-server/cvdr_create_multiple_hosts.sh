#!/bin/bash

set -e

usage() {
    echo "Usage: $0 [--num_hosts <num_hosts>] -- [<subcommands of cvdr create>]"
    echo "Prerequisites:"
    echo "  - Prepare accessible Cloud Orchestrator server"
    echo "  - Install cuttlefish-cvdremote to obtain cvdr binary"
    echo "  - Prepare a proper configuration file of cvdr"
}

NUM_HOSTS=1
while true; do
    case "$1" in
        -n|--num_hosts)
            NUM_HOSTS="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        --)
            shift
            break
            ;;
        *)
            echo "Invalid command" >&2
            usage
            exit 1
            ;;
    esac
done

cvdr create "$@" 2>&1 | sed "s/^/[Host 1] /"
if (( $NUM_HOSTS > 1 )); then
    for i in $(seq 2 $NUM_HOSTS); do
        cvdr create "$@" 2>&1 | sed "s/^/[Host $i] /" &
    done
    wait
fi
