# Building

From the top directory run:

```
go build ./... && go test ./...
```

To build the the executables run:

```
go build ./cmd/<executablename>
```

Where `<executablename>` is one of `cvdr` or `cloud_orchestrator`

# Running

When the cloud orchestrator runs it reads the conf.toml configuration file. The one provided with
the code uses *fakes* and *stubs* for modules that require a production environments. The OAuth2
module needs a file with the client configuration, which you can create with this command:

```
cat >../secrets.json << EOF
{
  "client_id": "",
  "client_secret": ""
}
```

> NOTE: The file is created outside the repository to allow a real OAuth2 client configuration to
be used if desired.

Then, running the cloud orchestrator is as easy as executing the `cloud_orchestrator` binary with no
arguments. It will listen on port 8080 for plain HTTP requests.

In development, the cloud orchestrator is configured to interact with a single host orchestrator
listening on 1081.

## The host orchestrator

> WARNING: The host orchestrator is developed in a different repository, so the following
instructions may become obsolete without notice.

1. Get The source code
    ```
    git clone https://github.com/google/android-cuttlefish
    ```

2. Build the host orchestrator binary:
    ```
    cd frontend/src/host_orchestrator
    go build ./...
    ```

3. Run the host orchestrator:

    ```
    HO_RUN_DIR=/tmp/host_orchestrator_runtime
    mkdir $HO_RUN_DIR

    ORCHESTRATOR_SOCKET_PATH=$HO_RUN_DIR/operator \
    ORCHESTRATOR_HTTP_PORT=1081 \
    ORCHESTRATOR_CVD_ARTIFACTS_DIR=$HO_RUN_DIR \
    ./host_orchestrator
    ```
