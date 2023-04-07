terraform {
  required_providers {
    coder = {
      source  = "coder/coder"
      version = "~> 0.7.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.55"
    }
  }
}

provider "coder" {
  feature_use_managed_variables = true
}

variable "ecs-cluster" {
  description = "Input the ECS cluster ARN to host the workspace"
}

data "coder_parameter" "cpu" {
  name         = "cpu"
  display_name = "CPU"
  description  = "The number of CPU units to reserve for the container"
  type         = "number"
  default      = "1024"
  mutable      = true
}

data "coder_parameter" "memory" {
  name         = "memory"
  display_name = "Memory"
  description  = "The amount of memory (in MiB) to allow the container to use"
  type         = "number"
  default      = "2048"
  mutable      = true
}

# configure AWS provider with creds present on Coder server host
provider "aws" {
  shared_config_files      = ["$HOME/.aws/config"]
  shared_credentials_files = ["$HOME/.aws/credentials"]
}

# coder workspace, created as an ECS task definition
resource "aws_ecs_task_definition" "workspace" {
  family = "coder"

  requires_compatibilities = ["EC2"]
  cpu                      = data.coder_parameter.cpu.value
  memory                   = data.coder_parameter.memory.value
  container_definitions = jsonencode([
    {
      name      = "coder-workspace-${data.coder_workspace.me.id}"
      image     = "codercom/enterprise-base:ubuntu"
      cpu       = tonumber(data.coder_parameter.cpu.value)
      memory    = tonumber(data.coder_parameter.memory.value)
      essential = true
      user      = "coder"
      command   = ["sh", "-c", coder_agent.coder.init_script]
      environment = [
        {
          "name"  = "CODER_AGENT_TOKEN"
          "value" = coder_agent.coder.token
        }
      ]
      mountPoints = [
        {
          # the name of the volume to mount
          sourceVolume = "home-dir-${data.coder_workspace.me.id}"
          # path on the container to mount the volume at
          containerPath = "/home/coder"
        }
      ]
      portMappings = [
        {
          containerPort = 80
          hostPort      = 80
        }
      ]
    }
  ])

  # workspace persistent volume definition
  volume {
    name = "home-dir-${data.coder_workspace.me.id}"

    docker_volume_configuration {
      # "shared" ensures that the disk is persisted upon workspace restart
      scope         = "shared"
      autoprovision = true
      driver        = "local"
    }
  }
}

resource "aws_ecs_service" "workspace" {
  name            = "workspace-${data.coder_workspace.me.id}"
  cluster         = var.ecs-cluster
  task_definition = aws_ecs_task_definition.workspace.arn
  # scale the service to zero when the workspace is stopped
  desired_count = data.coder_workspace.me.start_count
}

data "coder_workspace" "me" {}

resource "coder_agent" "coder" {
  arch                   = "amd64"
  auth                   = "token"
  os                     = "linux"
  dir                    = "/home/coder"
  login_before_ready     = false
  startup_script_timeout = 180
  startup_script         = <<-EOT
    set -e

    # install and start code-server
    curl -fsSL https://code-server.dev/install.sh | sh -s -- --method=standalone --prefix=/tmp/code-server --version 4.8.3
    /tmp/code-server/bin/code-server --auth none --port 13337 >/tmp/code-server.log 2>&1 &
  EOT

  metadata {
    key          = "cpu"
    display_name = "CPU Usage"
    interval     = 5
    timeout      = 5
    script       = <<-EOT
      #!/bin/bash
      set -e
      top -bn1 | grep "Cpu(s)" | awk '{print $2 + $4 "%"}'
    EOT
  }
  metadata {
    key          = "memory"
    display_name = "Memory Usage"
    interval     = 5
    timeout      = 5
    script       = <<-EOT
      #!/bin/bash
      set -e
      free -m | awk 'NR==2{printf "%.2f%%\t", $3*100/$2 }'
    EOT
  }
  metadata {
    key          = "disk"
    display_name = "Disk Usage"
    interval     = 600 # every 10 minutes
    timeout      = 30  # df can take a while on large filesystems
    script       = <<-EOT
      #!/bin/bash
      set -e
      df /home/coder | awk '$NF=="/"{printf "%s", $5}'
    EOT
  }
}

resource "coder_app" "code-server" {
  agent_id     = coder_agent.coder.id
  slug         = "code-server"
  display_name = "code-server"
  icon         = "/icon/code.svg"
  url          = "http://localhost:13337?folder=/home/coder"
  subdomain    = false
  share        = "owner"

  healthcheck {
    url       = "http://localhost:13337/healthz"
    interval  = 3
    threshold = 10
  }
}
