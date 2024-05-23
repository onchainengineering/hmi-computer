terraform {
  required_providers {
    coder = {
      source = "coder/coder"
    }
    docker = {
      source = "kreuzwerker/docker"
    }
  }
}

locals {
  username = data.coder_workspace.me.owner
}

data "coder_provisioner" "me" {
}

data "coder_workspace" "me" {
}

data "coder_workspace_tags" "custom_workspace_tags" {
  tags = {
    "zone"       = "developers"
    "os"         = data.coder_parameter.os_selector.value
    "project_id" = "PROJECT_${data.coder_parameter.project_name.value}"
    "cache"      = data.coder_parameter.feature_cache_enabled.value == "true" ? "with-cache" : "no-cache"
  }
}

data "coder_parameter" "os_selector" {
  name         = "os_selector"
  display_name = "OS runtime"
  default      = "linux"

  option {
    name  = "Linux"
    value = "linux"
  }
  option {
    name  = "OSX"
    value = "osx"
  }
  option {
    name  = "Windows"
    value = "windows"
  }

  mutable = false
}

data "coder_parameter" "project_name" {
  name         = "project_name"
  display_name = "Project name"
  description  = "Specify the project name."

  mutable = false
}

data "coder_parameter" "feature_cache_enabled" {
  name         = "feature_cache_enabled"
  display_name = "Enable cache?"
  type         = "bool"
  default      = false

  mutable = false
}

resource "coder_agent" "main" {
  arch           = data.coder_provisioner.me.arch
  os             = "linux"
  startup_script = <<EOF
    #!/bin/sh
    # install and start code-server
    curl -fsSL https://code-server.dev/install.sh | sh -s -- --version 4.8.3
    code-server --auth none --port 13337
    EOF

  env = {
    GIT_AUTHOR_NAME     = "${data.coder_workspace.me.owner}"
    GIT_COMMITTER_NAME  = "${data.coder_workspace.me.owner}"
    GIT_AUTHOR_EMAIL    = "${data.coder_workspace.me.owner_email}"
    GIT_COMMITTER_EMAIL = "${data.coder_workspace.me.owner_email}"
  }
}

resource "coder_app" "code-server" {
  agent_id     = coder_agent.main.id
  slug         = "code-server"
  display_name = "code-server"
  url          = "http://localhost:13337/?folder=/home/${local.username}"
  icon         = "/icon/code.svg"
  subdomain    = false
  share        = "owner"

  healthcheck {
    url       = "http://localhost:13337/healthz"
    interval  = 5
    threshold = 6
  }
}

resource "docker_volume" "home_volume" {
  name = "coder-${data.coder_workspace.me.id}-home"
  lifecycle {
    ignore_changes = all
  }
  labels {
    label = "coder.owner"
    value = data.coder_workspace.me.owner
  }
  labels {
    label = "coder.owner_id"
    value = data.coder_workspace.me.owner_id
  }
  labels {
    label = "coder.workspace_id"
    value = data.coder_workspace.me.id
  }
  labels {
    label = "coder.workspace_name_at_creation"
    value = data.coder_workspace.me.name
  }
}

resource "coder_metadata" "home_info" {
  resource_id = docker_volume.home_volume.id

  item {
    key   = "size"
    value = "5 GiB"
  }
}

resource "docker_container" "workspace" {
  count      = data.coder_workspace.me.start_count
  image      = "ubuntu:22.04"
  name       = "coder-${data.coder_workspace.me.owner}-${lower(data.coder_workspace.me.name)}"
  hostname   = data.coder_workspace.me.name
  entrypoint = ["sh", "-c", replace(coder_agent.main.init_script, "/localhost|127\\.0\\.0\\.1/", "host.docker.internal")]
  env = [
    "CODER_AGENT_TOKEN=${coder_agent.main.token}",
  ]
  host {
    host = "host.docker.internal"
    ip   = "host-gateway"
  }
  volumes {
    container_path = "/home/${local.username}"
    volume_name    = docker_volume.home_volume.name
    read_only      = false
  }

  labels {
    label = "coder.owner"
    value = data.coder_workspace.me.owner
  }
  labels {
    label = "coder.owner_id"
    value = data.coder_workspace.me.owner_id
  }
  labels {
    label = "coder.workspace_id"
    value = data.coder_workspace.me.id
  }
  labels {
    label = "coder.workspace_name"
    value = data.coder_workspace.me.name
  }
}
