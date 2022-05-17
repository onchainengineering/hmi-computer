terraform {
  required_providers {
    coder = {
      source  = "coder/coder"
      version = "0.4.1"
    }
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 2.16.0"
    }
  }
}

# Admin parameters

# Comment this out if you are specifying a different docker
# host on the "docker" provider below.
variable "step1_docker_host_warning" {
  description = <<-EOF
  This template will use the Docker socket present on
  the Coder host, which is not necessarily your local machine.

  You can specify a different host in the template file and
  surpress this warning.
  EOF
  validation {
    condition     = contains(["Continue using /var/run/docker.sock on the Coder host"], var.step1_docker_host_warning)
    error_message = "Cancelling template create."
  }

  sensitive = true
}
variable "step2_arch" {
  description = <<-EOF
  arch: What archicture is your Docker host on?

  note: codercom/enterprise-* images are only built for amd64
  EOF

  validation {
    condition     = contains(["amd64", "arm64", "armv7"], var.step2_arch)
    error_message = "Value must be amd64, arm64, or armv7."
  }
  sensitive = true
}

provider "docker" {
  host = "unix:///var/run/docker.sock"
}

provider "coder" {
  url = "http://host.docker.internal:7080"
}

data "coder_workspace" "me" {
}

resource "coder_agent" "dev" {
  arch = var.step2_arch
  os   = "linux"
}

variable "docker_image" {
  description = "What docker image would you like to use for your workspace?"
  # The codercom/enterprise-* images are only built for amd64
  default = "codercom/enterprise-base:ubuntu"
  validation {
    condition     = contains(["codercom/enterprise-base:ubuntu", "codercom/enterprise-node:ubuntu", "codercom/enterprise-intellij:ubuntu"], var.docker_image)
    error_message = "Invalid Docker Image!"
  }

  validation {
    condition     = contains(["codercom/enterprise-base:ubuntu", "codercom/enterprise-node:ubuntu", "codercom/enterprise-intellij:ubuntu"], var.docker_image)
    error_message = "Invalid Docker Image!"
  }

}

resource "docker_volume" "home_volume" {
  name = "coder-${data.coder_workspace.me.owner}-${data.coder_workspace.me.name}-root"
}

resource "docker_container" "workspace" {
  count = data.coder_workspace.me.start_count
  image = var.docker_image
  # Uses lower() to avoid Docker restriction on container names.
  name = "coder-${data.coder_workspace.me.owner}-${lower(data.coder_workspace.me.name)}"
  # Hostname makes the shell more user friendly: coder@my-workspace:~$
  hostname = lower(data.coder_workspace.me.name)
  dns      = ["1.1.1.1"]
  command  = ["sh", "-c", coder_agent.dev.init_script]
  env      = ["CODER_AGENT_TOKEN=${coder_agent.dev.token}"]
  host {
    host = "host.docker.internal"
    ip   = "host-gateway"
  }
  volumes {
    container_path = "/home/coder/"
    volume_name    = docker_volume.home_volume.name
    read_only      = false
  }
}
