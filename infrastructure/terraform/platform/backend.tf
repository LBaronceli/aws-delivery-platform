terraform {
  required_version = ">= 1.10.0"

  backend "s3" {
    key          = "delivery-platform/platform/terraform.tfstate"
    region       = "ap-southeast-2"
    encrypt      = true
    use_lockfile = true
  }
}
