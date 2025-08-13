# Cloud orchestrator on GCE instance

This page describes how to run cloud orchestrator at GCE instance.

## Prepare GCE image

Execute following command to retrieve GCE image for x86_64 instance.

```bash
scripts/on-premises/gcp/gce_image_builder.sh -p $IMAGE_PROJECT
```

## Launch GCE instance with the image

For GCE x86_64 bare-metal instance, execute command like below.
```bash
gcloud --project $PROJECT compute instances create $INSTANCE_NAME \
    --boot-disk-size=$DISK_SIZE_GB \
    --can-ip-forward \
    --image-family=cf-cloud-orchestrator-amd64 \
    --image-project=$IMAGE_PROJECT \
    --machine-type=c3-highcpu-192-metal \
    --maintenance-policy=TERMINATE \
    --tags=http-server,https-server,lb-health-check \
    --zone=$ZONE
```

For GCE x86_64 VM instance, execute command like below.
```bash
gcloud --project $PROJECT compute instances create $INSTANCE_NAME \
    --boot-disk-size=$DISK_SIZE_GB \
    --can-ip-forward \
    --enable-nested-virtualization \
    --image-family=cf-cloud-orchestrator-amd64 \
    --image-project=$IMAGE_PROJECT \
    --machine-type=n1-standard-8 \
    --tags=http-server,https-server,lb-health-check \
    --zone=$ZONE
```

After the instance is booted, Cloud orchestrator will be running on the port
8080.
