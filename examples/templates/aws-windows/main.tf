terraform {
  required_providers {
    coder = {
      source  = "coder/coder"
      version = "0.3.4"
    }
  }
}

variable "access_key" {
  description = <<EOT
Create an AWS access key to provision resources with Coder:
- https://console.aws.amazon.com/iam/home#/users

AWS Access Key
EOT
  sensitive   = true
}

variable "secret_key" {
  description = <<EOT
AWS Secret Key
EOT
  sensitive   = true
}

# Last updated 2022-05-31
# aws ec2 describe-regions | jq -r '[.Regions[].RegionName] | sort'
variable "region" {
  description = "What region should your workspace live in?"
  default     = "us-east-1"
  validation {
    condition     = contains([
      "ap-northeast-1",
      "ap-northeast-2",
      "ap-northeast-3",
      "ap-south-1",
      "ap-southeast-1",
      "ap-southeast-2",
      "ca-central-1",
      "eu-central-1",
      "eu-north-1",
      "eu-west-1",
      "eu-west-2",
      "eu-west-3",
      "sa-east-1",
      "us-east-1",
      "us-east-2",
      "us-west-1",
      "us-west-2"
    ], var.region)
    error_message = "Invalid region!"
  }
}

provider "aws" {
  region     = var.region
  access_key = var.access_key
  secret_key = var.secret_key
}

data "coder_workspace" "me" {
}

data "aws_ami" "windows" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["Windows_Server-2019-English-Full-Base-*"]
  }
}

resource "coder_agent" "dev" {
  arch = "amd64"
  auth = "aws-instance-identity"
  os   = "windows"
}

locals {

  # User data is used to stop/start AWS instances. See:
  # https://github.com/hashicorp/terraform-provider-aws/issues/22

  user_data_start = <<EOT
<powershell>
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
${coder_agent.dev.init_script}
</powershell>
<persist>true</persist>
EOT

  user_data_end = <<EOT
<powershell>
shutdown /s
</powershell>
<persist>true</persist>
EOT
}

resource "aws_instance" "dev" {
  ami               = data.aws_ami.windows.id
  availability_zone = "${var.region}a"
  instance_type     = "t3.micro"
  count             = 1

  user_data = data.coder_workspace.me.transition == "start" ? local.user_data_start : local.user_data_end
  tags = {
    Name = "coder-${data.coder_workspace.me.owner}-${data.coder_workspace.me.name}"
    # Required if you are using our example policy, see template README
    Coder_Provisioned = "true"
  }

}
