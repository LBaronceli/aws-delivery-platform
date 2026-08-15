# AWS Delivery Platform

A small delivery-tracking API built with Go and deployed to AWS ECS Fargate
with Terraform. The project is a practical environment for developing an
AWS-native service architecture incrementally, without letting the sample
application dominate the infrastructure work.

## Current status

Phase 4 is complete. The project currently provides:

- A Go HTTP API for creating, listing, retrieving, and updating deliveries
- A concurrent-safe in-memory delivery store
- A multi-stage, non-root Docker image
- A private ECS Fargate service behind a public Application Load Balancer
- A custom two-AZ VPC with public and private subnets
- A persistent ECR repository for application images
- An account-level USD 10 budget with email alerts
- Three isolated Terraform states for foundation, platform, and app resources

Delivery data is intentionally in memory for now. It is lost when the API task
restarts and is not shared between multiple tasks. Aurora PostgreSQL is the
next planned phase.

## Architecture

```text
Allowed developer IP
        |
        | HTTP :80
        v
Public Application Load Balancer
        |
        | HTTP :8080
        v
ECS Fargate task in private subnets
        |
        | outbound through NAT
        v
ECR and CloudWatch Logs
```

The ALB spans two public subnets and is reachable only from the `/32` CIDR in
the app variables. The ECS task spans two private subnets, has no public IP,
and accepts traffic only from the ALB security group. This is a development
environment: the endpoint currently uses HTTP and does not have authentication.

Terraform is split by lifecycle:

| Root | Purpose | Normal lifecycle |
| --- | --- | --- |
| `foundation` | Account budget and billing notifications | Persistent |
| `platform` | ECR repository and container images | Persistent |
| `app` | VPC, NAT, ALB, ECS, IAM, and logs | Disposable |

## Repository layout

```text
app/
├── cmd/api/                         API entry point
├── internal/api/                    HTTP routes and tests
├── internal/delivery/               Domain model and in-memory store
└── Dockerfile

infrastructure/terraform/
├── foundation/                      Account-level controls
├── platform/                        Persistent application artifacts
└── app/                             Runtime application infrastructure

temp/                                Local planning notes
```

## API

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/health` | Service health check |
| `POST` | `/deliveries` | Create a delivery |
| `GET` | `/deliveries` | List deliveries in creation order |
| `GET` | `/deliveries/{id}` | Retrieve a delivery |
| `PATCH` | `/deliveries/{id}` | Update pickup, destination, or status |

Create a delivery:

```http
POST /deliveries
Content-Type: application/json

{
  "pickup": "Wellington",
  "destination": "Napier"
}
```

Example response:

```json
{
  "id": "delivery-fea93fe751cbb580",
  "pickup": "Wellington",
  "destination": "Napier",
  "status": "created"
}
```

Update a delivery:

```http
PATCH /deliveries/delivery-fea93fe751cbb580
Content-Type: application/json

{
  "status": "scheduled"
}
```

Valid statuses are:

```text
created
scheduled
picked_up
in_transit
delivered
cancelled
```

The API rejects unknown JSON fields, malformed bodies, empty locations, and
invalid statuses with `400 Bad Request`. Missing delivery IDs return
`404 Not Found`.

## Local development

### Prerequisites

- Go 1.26 or later
- Docker Desktop

Run the API:

```bash
cd app
go run ./cmd/api
```

The server listens on port `8080` by default. Override it with the `PORT`
environment variable.

```bash
curl http://localhost:8080/health

curl http://localhost:8080/deliveries \
  -H 'Content-Type: application/json' \
  -d '{"pickup":"Wellington","destination":"Napier"}'

curl http://localhost:8080/deliveries
```

Run the test suite:

```bash
cd app
go test ./...
go test -race ./...
```

### Docker

Build and run the container locally:

```bash
docker build -t delivery-api:local app
docker run --rm -p 8080:8080 delivery-api:local
```

Verify it from another terminal:

```bash
curl http://localhost:8080/health
```

## Deploy to AWS

### Prerequisites

- Terraform 1.10 or later
- AWS CLI v2
- Docker Desktop
- AWS credentials with permission to create the configured resources
- An S3 bucket for remote Terraform state

Authenticate and set the deployment region:

```bash
export AWS_REGION=ap-southeast-2
aws sts get-caller-identity
```

### 1. Create the state bucket

The state bucket is a bootstrap dependency and is deliberately not managed by
these Terraform roots. If it already exists, skip its creation and set
`TF_STATE_BUCKET` to its name.

Bucket names are globally unique, so choose a name available to your account:

```bash
export AWS_ACCOUNT_ID="$(
  aws sts get-caller-identity --query Account --output text
)"
export TF_STATE_BUCKET="delivery-platform-tfstate-${AWS_ACCOUNT_ID}"

aws s3api create-bucket \
  --bucket "${TF_STATE_BUCKET}" \
  --region "${AWS_REGION}" \
  --create-bucket-configuration "LocationConstraint=${AWS_REGION}"

aws s3api put-bucket-versioning \
  --bucket "${TF_STATE_BUCKET}" \
  --versioning-configuration Status=Enabled

aws s3api put-bucket-encryption \
  --bucket "${TF_STATE_BUCKET}" \
  --server-side-encryption-configuration \
  '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"AES256"}}]}'

aws s3api put-public-access-block \
  --bucket "${TF_STATE_BUCKET}" \
  --public-access-block-configuration \
  BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true
```

This repository currently uses `delivery-platform-tfstate`. Use that existing
name when working with the current AWS account:

```bash
export TF_STATE_BUCKET=delivery-platform-tfstate
```

### 2. Initialize Terraform

Each root uses the same bucket with a different state key:

| Root | State key |
| --- | --- |
| Foundation | `delivery-platform/foundation/terraform.tfstate` |
| Platform | `delivery-platform/platform/terraform.tfstate` |
| App | `delivery-platform/dev/terraform.tfstate` |

```bash
terraform -chdir=infrastructure/terraform/foundation init \
  -backend-config="bucket=${TF_STATE_BUCKET}"

terraform -chdir=infrastructure/terraform/platform init \
  -backend-config="bucket=${TF_STATE_BUCKET}"

terraform -chdir=infrastructure/terraform/app init \
  -backend-config="bucket=${TF_STATE_BUCKET}"
```

The S3 backend uses native state locking. Commit each root's
`.terraform.lock.hcl`; do not commit `.terraform/`, state files, plans, or real
variable files.

### 3. Configure variables

Create local variable files from the tracked examples:

```bash
cp infrastructure/terraform/foundation/terraform.tfvars.example \
  infrastructure/terraform/foundation/terraform.tfvars

cp infrastructure/terraform/platform/terraform.tfvars.example \
  infrastructure/terraform/platform/terraform.tfvars

cp infrastructure/terraform/app/terraform.tfvars.example \
  infrastructure/terraform/app/terraform.tfvars
```

Update:

- `foundation/terraform.tfvars`: set `billing_alert_email`; the budget defaults
  to USD 10 per month.
- `app/terraform.tfvars`: set `allowed_cidr` to your public IP with `/32` and,
  after pushing an image, set `image_uri` to its complete ECR URI and tag.
- `platform/terraform.tfvars`: defaults are suitable for the development
  environment unless the region or naming changes.

Find your current public IP with:

```bash
curl https://checkip.amazonaws.com
```

Real `terraform.tfvars` files are ignored by Git.

### 4. Create foundation and platform resources

Apply the budget first:

```bash
terraform -chdir=infrastructure/terraform/foundation plan \
  -out=foundation.tfplan
terraform -chdir=infrastructure/terraform/foundation apply \
  foundation.tfplan
```

The budget sends actual-spend alerts at 30%, 50%, 80%, and 100%, plus a
forecast alert at 80%. AWS Budgets is an alerting mechanism, not a hard spending
limit.

Create the persistent ECR repository:

```bash
terraform -chdir=infrastructure/terraform/platform plan \
  -out=platform.tfplan
terraform -chdir=infrastructure/terraform/platform apply \
  platform.tfplan
```

### 5. Build and push the image

Get the ECR URL and authenticate Docker:

```bash
export ECR_REPOSITORY_URL="$(
  terraform -chdir=infrastructure/terraform/platform output \
    -raw ecr_repository_url
)"
export ECR_REGISTRY="${ECR_REPOSITORY_URL%%/*}"

aws ecr get-login-password --region "${AWS_REGION}" |
  docker login --username AWS --password-stdin "${ECR_REGISTRY}"
```

Tags are immutable, so use a unique tag for every image:

```bash
export IMAGE_TAG="$(git rev-parse --short HEAD)-$(date -u +%Y%m%d%H%M%S)"
export IMAGE_URI="${ECR_REPOSITORY_URL}:${IMAGE_TAG}"

docker build \
  --platform linux/amd64 \
  --tag "${IMAGE_URI}" \
  app

docker push "${IMAGE_URI}"
echo "${IMAGE_URI}"
```

Copy the printed URI into `image_uri` in
`infrastructure/terraform/app/terraform.tfvars`.

### 6. Deploy the application stack

```bash
terraform -chdir=infrastructure/terraform/app plan \
  -out=deployment.tfplan
terraform -chdir=infrastructure/terraform/app apply \
  deployment.tfplan
```

Retrieve the ALB endpoint and verify the service:

```bash
export API_URL="$(
  terraform -chdir=infrastructure/terraform/app output -raw api_url
)"

curl "${API_URL}/health"

curl "${API_URL}/deliveries" \
  -H 'Content-Type: application/json' \
  -d '{"pickup":"Wellington","destination":"Napier"}'

curl "${API_URL}/deliveries"
```

Tail application logs:

```bash
aws logs tail /ecs/delivery-platform-dev-api \
  --region "${AWS_REGION}" \
  --follow
```

## Updating the service

For an application change:

1. Run the Go tests.
2. Build and push a new immutable image tag.
3. Update `image_uri` in the app variable file.
4. Plan and apply the app root.

ECS registers a new task-definition revision and performs a rolling deployment
behind the existing target group.

## Teardown

The ALB, NAT Gateway, and Fargate task incur charges while provisioned. For a
normal end-of-session teardown, destroy only the disposable app root:

```bash
terraform -chdir=infrastructure/terraform/app destroy
```

This preserves the budget, ECR repository, and images, making the next session
faster. Recreate the app with another plan and apply.

The foundation budget and platform repository have Terraform
`prevent_destroy` guards. A full teardown requires intentionally removing those
guards first. A non-empty ECR repository also requires `force_delete = true`.
The S3 state bucket remains separate and must be deleted manually if it is ever
no longer required.

## Troubleshooting

- `503 Service Unavailable`: wait for ECS to register a healthy target, then
  inspect the target group health and CloudWatch logs.
- Request timeout: confirm your current public IP matches `allowed_cidr` in the
  app variable file and reapply if it changed.
- ECR tag already exists: tags are immutable; build with a new tag.
- ECS cannot pull the image: confirm `image_uri` exists in ECR and includes the
  tag, and verify that the private subnet has NAT connectivity.

## Roadmap

- Aurora PostgreSQL persistence and migrations
- Delivery status history
- SQS-backed delivery processing worker
- EventBridge lifecycle events
- API Gateway with VPC Link
- Auth0 authentication and user-scoped deliveries
- Frontend and custom domain
