# Steps

## STEP 1: Prepare `terraform.tfvars`

Look at the `variables.tf` and provide necessary variables and/or override defaults via `terraform.tfvars`.

## STEP 2: Set up project

```
$ cd terraform
$ terraform plan
$ terraform apply
```

`terraform apply` creates a `build-and-deploy.sh` which you need to run after `terraform apply` ends successfully.

## STEP 3: Deploy Cloud Run service

```
$ ./build-and-deploy.sh
```
