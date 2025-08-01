# Activate cloud orchestrator at on-premise server

This page describes how to run cloud orchestrator at on-premise server, which
manages docker instances containing the host orchestrator inside.

Note that this is under development, some features may be broken yet.
Please let us know if you faced at any bugs.

## Try cloud orchestrator

Currently we're hosting docker images and its configuration files in Artifact
Registry.
Please execute the commands below if you want to download and run the cloud
orchestrator.

Also, please choose one location among `us`, `europe`, or `asia`.
It's available to download artifacts from any location, but download latency is
different based on your location.

```bash
DOWNLOAD_LOCATION=us # Choose one among us, europe, or asia.
docker pull $DOWNLOAD_LOCATION-docker.pkg.dev/android-cuttlefish-artifacts/cuttlefish-orchestration/cuttlefish-cloud-orchestrator
wget -O conf.toml https://artifactregistry.googleapis.com/download/v1/projects/android-cuttlefish-artifacts/locations/$DOWNLOAD_LOCATION/repositories/cloud-orchestrator-config/files/on-premise-single-server:main:conf.toml:download?alt=media
docker run \
    -p 8080:8080 \
    -e CONFIG_FILE="/conf.toml" \
    -v $PWD/conf.toml:/conf.toml \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -t $DOWNLOAD_LOCATION-docker.pkg.dev/android-cuttlefish-artifacts/cuttlefish-orchestration/cuttlefish-cloud-orchestrator:latest
```

To enable TURN server support for WebRTC peer-to-peer connections, configure
your TURN server settings in the `conf.toml` file before starting the cloud
orchestrator.
See the example below.
```
[[WebRTC.IceServers]]
URLs = ["turn:localhost:3478"]
Username = "username"
Credential = "credential"
```

If there's a firewall which blocks accessing cloud orchestrator with HTTP/HTTPS
requests, please try using SOCKS5 proxy. Establishing SOCKS5 proxy by creating
SSH dynamic port forwarding is available with following command.
```bash
ssh -D ${SOCKS5_PORT} -q -C -N ${USERNAME}@${CLOUD_ORCHESTRATOR_IPv4_ADDRESS}
```
## Use cloud orchestrator by cvdr

The config file for `cvdr` is located at
[scripts/on-premises/single-server/cvdr.toml](cvdr.toml).
Please follow the steps at [cvdr.md](/docs/cvdr.md), to get started with `cvdr`.

### Batch creation by cvdr

Unfortunately, we don't support bulk/batch creation on `cvdr` yet, such as
creating Cuttlefish instances across multiple hosts.
Please
[scripts/on-premises/single-server/cvdr_create_multiple_hosts.sh](cvdr_create_multiple_hosts.sh)
instead for a moment to create multiple identical hosts with running Cuttlefish
instances in them.

## Manually build and run cloud orchestrator

The config file for cloud orchestrator is at
[scripts/on-premises/single-server/conf.toml](conf.toml).
Please follow the steps at [cloud_orchestrator.md](/docs/cloud_orchestrator.md),
to build and run cloud orchestrator.

Also, you may need to prepare another docker image containing the host
orchestrator inside, unlike steps in
[Try cloud orchestrator](#try-cloud-orchestrator).
Please follow the docker part of
[README.md](https://github.com/google/android-cuttlefish/blob/main/README.md#docker)
in `google/android-cuttlefish` github repository, and check if proper docker
image exists via `docker image list`.
