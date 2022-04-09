terraform {
  required_providers {
    coder = {
      source  = "coder/coder"
      version = "~> 0.3.1"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.10"
    }
  }
}

provider "kubernetes" {
  config_path = "~/.kube/config"
}

data "coder_workspace" "me" {}

resource "coder_agent" "go" {
  os   = "linux"
  arch = "amd64"
}

resource "coder_agent" "java" {
  os   = "linux"
  arch = "amd64"
}

resource "coder_agent" "ubuntu" {
  os   = "linux"
  arch = "amd64"
}

resource "kubernetes_pod" "main" {
  count = data.coder_workspace.me.start_count
  metadata {
    name = "coder-${data.coder_workspace.me.owner}-${data.coder_workspace.me.name}"
  }
  spec {
    container {
      name    = "go"
      image   = "mcr.microsoft.com/vscode/devcontainers/go:1"
      command = ["sh", "-c", coder_agent.go.init_script]
      security_context {
        run_as_user = "1000"
      }
      env {
        name  = "CODER_TOKEN"
        value = coder_agent.go.token
      }
    }
    container {
      name    = "java"
      image   = "mcr.microsoft.com/vscode/devcontainers/java"
      command = ["sh", "-c", coder_agent.java.init_script]
      security_context {
        run_as_user = "1000"
      }
      env {
        name  = "CODER_TOKEN"
        value = coder_agent.java.token
      }
    }
    container {
      name    = "ubuntu"
      image   = "mcr.microsoft.com/vscode/devcontainers/base:ubuntu"
      command = ["sh", "-c", coder_agent.ubuntu.init_script]
      security_context {
        run_as_user = "1000"
      }
      env {
        name  = "CODER_TOKEN"
        value = coder_agent.ubuntu.token
      }
    }
  }
}

