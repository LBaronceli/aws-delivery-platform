# AWS Delivery Platform

A small delivery-tracking platform for learning AWS networking, ECS/Fargate,
queues, events, authentication, and deployment.

The current development deployment is:

```text
Developer IP (/32)
        |
        | TCP 8080
        v
Security group
        |
        v
ECS service -> Fargate task -> ECR image
                       |
                       v
                 CloudWatch Logs
```

The Fargate task currently runs in the default VPC with a public IP. Direct
access is restricted to the CIDR configured in `terraform.tfvars`. This is a
temporary development setup; the task will move behind an Application Load
Balancer and API Gateway later.

## Repository layout

```text
app/                        Go API and Dockerfile
infrastructure/terraform/   AWS infrastructure
temp/                       Local planning notes
```

## Prerequisites

- Go 1.26 or later
- Docker Desktop
- Terraform 1.10 or later
- AWS CLI v2 authenticated to the target AWS account

The examples use the Sydney region:

```bash
export AWS_REGION=ap-southeast-2
aws sts get-caller-identity
```

## Run the API locally

```bash
cd app
go run ./cmd/api
```

The API listens on port `8080` by default. Set `PORT` to use a different port.

```bash
curl http://localhost:8080/health
curl http://localhost:8080/hello
```

## Run with Docker

Build from the `app` directory, which contains the Dockerfile and its build
context:

```bash
cd app
docker build -t delivery-api:local .
docker run --rm -p 8080:8080 delivery-api:local
```

From another terminal:

```bash
curl http://localhost:8080/health
```

The health response is:

```json
{"status":"ok"}
```

## Test

```bash
cd app
go test ./...
```

## Terraform state backend

Terraform stores development state at this key in a private S3 bucket:

```text
delivery-platform/dev/terraform.tfstate
```

The backend uses S3-native locking. The state bucket must exist before running
`terraform init`, and should have versioning, encryption, and all public access
blocked.

Set the bucket name for your AWS account and initialize the backend:

```bash
export AWS_ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"
export TF_STATE_BUCKET="delivery-platform-tfstate-${AWS_ACCOUNT_ID}-${AWS_REGION}"

terraform -chdir=infrastructure/terraform init \
  -backend-config="bucket=${TF_STATE_BUCKET}"
```

Terraform caches the backend configuration under `.terraform`, so the bucket
argument is only needed again when reinitializing the backend or using a fresh
checkout.

## Configure Terraform inputs

Create the local variable file once:

```bash
cp infrastructure/terraform/terraform.tfvars.example \
  infrastructure/terraform/terraform.tfvars
```

Set `allowed_cidr` to your current public IP with a `/32` suffix:

```bash
curl https://checkip.amazonaws.com
```

Also set `image_uri` to the complete immutable ECR image URI, including its
tag. The real `terraform.tfvars` is ignored by Git; keep
`terraform.tfvars.example` updated with safe example values.

## First deployment

There is a first-deployment dependency: ECR must exist before an image can be
pushed, and the image must exist before the ECS service can start successfully.

Create only the ECR repository the first time:

```bash
terraform -chdir=infrastructure/terraform apply \
  -target=aws_ecr_repository.api
```

Targeted applies are only used for this initial bootstrap. Normal deployments
should use a complete plan.

Get the repository URL and authenticate Docker:

```bash
export ECR_REPOSITORY_URL="$(
  terraform -chdir=infrastructure/terraform output -raw ecr_repository_url
)"
export ECR_REGISTRY="${ECR_REPOSITORY_URL%%/*}"

aws ecr get-login-password --region "${AWS_REGION}" |
  docker login --username AWS --password-stdin "${ECR_REGISTRY}"
```

Build and push a uniquely tagged x86 image from the repository root:

```bash
export IMAGE_TAG="$(git rev-parse --short HEAD)"
export IMAGE_URI="${ECR_REPOSITORY_URL}:${IMAGE_TAG}"

docker build \
  --platform linux/amd64 \
  --tag "${IMAGE_URI}" \
  app

docker push "${IMAGE_URI}"
```

ECR tags are immutable in this project. Do not reuse a tag for a different
image. Copy the value printed by `echo "${IMAGE_URI}"` into the `image_uri`
entry in `terraform.tfvars`.

Create or update all infrastructure:

```bash
terraform -chdir=infrastructure/terraform fmt -check
terraform -chdir=infrastructure/terraform validate
terraform -chdir=infrastructure/terraform plan -out=deployment.tfplan
terraform -chdir=infrastructure/terraform apply deployment.tfplan
```

The configuration creates:

- An encrypted ECR repository with immutable tags and scan-on-push
- An ECS cluster, task definition, and one-task Fargate service
- An ECS task execution role for ECR pulls and CloudWatch logging
- A CloudWatch log group with seven-day retention
- A security group allowing port `8080` only from `allowed_cidr`

## Reach the deployed API

The task's public IP can change whenever ECS replaces it. Retrieve the current
address with the AWS CLI:

```bash
export ECS_CLUSTER=delivery-platform-dev
export ECS_SERVICE=delivery-platform-dev-api

export ECS_TASK_ARN="$(
  aws ecs list-tasks \
    --cluster "${ECS_CLUSTER}" \
    --service-name "${ECS_SERVICE}" \
    --query 'taskArns[0]' \
    --output text
)"

export ECS_ENI_ID="$(
  aws ecs describe-tasks \
    --cluster "${ECS_CLUSTER}" \
    --tasks "${ECS_TASK_ARN}" \
    --query 'tasks[0].attachments[0].details[?name==`networkInterfaceId`].value | [0]' \
    --output text
)"

export ECS_PUBLIC_IP="$(
  aws ec2 describe-network-interfaces \
    --network-interface-ids "${ECS_ENI_ID}" \
    --query 'NetworkInterfaces[0].Association.PublicIp' \
    --output text
)"

curl "http://${ECS_PUBLIC_IP}:8080/health"
```

Tail the application logs with:

```bash
aws logs tail /ecs/delivery-platform-dev-api \
  --region "${AWS_REGION}" \
  --follow
```

If your public IP changes, update `allowed_cidr` in `terraform.tfvars`, then
run another complete Terraform plan and apply.
