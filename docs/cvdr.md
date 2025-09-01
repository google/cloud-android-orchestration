# cvdr

This page describes about `cvdr` and its usage.

## What's cvdr?

`cvdr` is a CLI binary tool for accessing and managing Cuttlefish instances
remotely.
It wraps [Cloud Orchestrator](cloud_orchestrator.md), to provide user-friendly
interface.

Please run `cvdr --help` for advanced functionalities of `cvdr` not described
below, such as launching Cuttlefish with locally built image.

## Download cvdr

`cuttlefish-cvdremote` is available to download via `apt install` with adding
the apt repository at Artifact Registry.
```bash
curl https://us-apt.pkg.dev/doc/repo-signing-key.gpg | sudo apt-key add -
echo 'deb https://us-apt.pkg.dev/projects/android-cuttlefish-artifacts android-cuttlefish-nightly main' | \
  sudo tee -a /etc/apt/sources.list.d/artifact-registry.list
sudo apt update
sudo apt install cuttlefish-cvdremote
cvdr --help
```

## Configure cvdr

Please check and modify the configuration file(`~/.config/cvdr/cvdr.toml`).
See either 
[build/debian/cuttlefish_cvdremote/host/etc/cvdr.toml](/build/debian/cuttlefish_cvdremote/host/etc/cvdr.toml)
or
[scripts/on-premises/single-server/cvdr.toml](/scripts/on-premises/single-server/cvdr.toml)
as examples of how to write a configuration file.

## Use cvdr

Let's assume using the latest Cuttlefish x86_64 image enrolled in
[ci.android.com](https://ci.android.com/).

Please run:
```bash
cvdr \
--branch=aosp-main \
--build_target=aosp_cf_x86_64_phone-trunk_staging-userdebug \
create
```

Then we expect the result like below.
```
Creating Host........................................ OK
Fetching main bundle artifacts....................... OK
Starting and waiting for boot complete............... OK
Connecting to cvd-1.................................. OK
2e8137432a96f93558c838da5e590ec775a97e5a7bb20e66929d1a59eb337351 (http://localhost:8080/v1/zones/local/hosts/2e8137432a96f93558c838da5e590ec775a97e5a7bb20e66929d1a59eb337351/)
  cvd/1
  Status: Running
  ADB: 127.0.0.1:33975
  Displays: [720 x 1280 ( 320 )]
  Logs: http://localhost:8080/v1/zones/local/hosts/2e8137432a96f93558c838da5e590ec775a97e5a7bb20e66929d1a59eb337351/cvds/1/logs/
```
If you want to validate, please refer the first provided URL in the output log
and check if the page seems like below.
Also, you should be able to see the device is enrolled via `adb devices`.
![cvdr_cf_creation](resources/cvdr_cf_creation_example.png)

## Manually build and run cvdr

To build `cvdr` manually, please run:
```bash
git clone https://github.com/google/cloud-android-orchestration.git
cd cloud-android-orchestration # Root directory of git repository
go build ./cmd/cvdr
```

To run `cvdr`, please run:
```bash
CVDR_USER_CONFIG_PATH=/path/to/cvdr.toml ./cvdr --help
```
