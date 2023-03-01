terraform {
  required_providers {
    coder = {
      source  = "coder/coder"
      version = "~> 0.6.14"
    }
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }
}



data "coder_parameter" "step1_do_project_id" {
  name        = "Enter project ID"
  description = <<-EOF
    Enter project ID

      $ doctl projects list
  EOF
  type        = "string"
  default     = "0"
  mutable     = false
  validation {
    # make sure length of alphanumeric string is 36
    regex = "^[a-zA-Z0-9]{36}$"
    error = "Invalid Digital Ocean Project ID."
  }
}

data "coder_parameter" "step2_do_admin_ssh_key" {
  name        = "Enter admin SSH key ID (some Droplet images require an SSH key to be set):"
  description = <<-EOF
    Can be set to "0" for no key.

    Note: Setting this to zero will break Fedora images and notify root passwords via email.

      $ doctl compute ssh-key list
  EOF
  type        = "number"
  default     = "0"
  mutable     = false
  validation {
    min = 0
    max = 999999
  }
}

data "coder_parameter" "droplet_image" {
  name    = "Which Droplet image would you like to use for your workspace?"
  default = "ubuntu-22-04-x64"
  type    = "string"
  mutable = false
  option {
    name  = "Ubuntu 22.04"
    value = "ubuntu-22-04-x64"
  }
  option {
    name  = "Ubuntu 20.04"
    value = "ubuntu-20-04-x64"
  }
  option {
    name  = "Fedora 36"
    value = "fedora-36-x64"
  }
  option {
    name  = "Fedora 35"
    value = "fedora-35-x64"
  }
  option {
    name  = "Debian 11"
    value = "debian-11-x64"
  }
  option {
    name  = "Debian 10"
    value = "debian-10-x64"
  }
  option {
    name  = "CentOS Stream 9"
    value = "centos-stream-9-x64"
  }
  option {
    name  = "CentOS Stream 8"
    value = "centos-stream-8-x64"
  }
  option {
    name  = "Rocky Linux 8"
    value = "rockylinux-8-x64"
  }
  option {
    name  = "Rocky Linux 8.4"
    value = "rockylinux-8-4-x64"
  }
}

data "coder_parameter" "droplet_size" {
  name    = "Which Droplet configuration would you like to use?"
  default = "s-1vcpu-1gb"
  type    = "string"
  mutable = false
  option {
    name  = "s-1vcpu-1gb"
    value = "s-1vcpu-1gb"
  }
  option {
    name  = "s-1vcpu-2gb"
    value = "s-1vcpu-2gb"
  }
  option {
    name  = "s-2vcpu-2gb"
    value = "s-2vcpu-2gb"
  }
  option {
    name  = "s-2vcpu-4gb"
    value = "s-2vcpu-4gb"
  }
  option {
    name  = "s-4vcpu-8gb"
    value = "s-4vcpu-8gb"
  }
  option {
    name  = "s-8vcpu-16gb"
    value = "s-8vcpu-16gb"
  }
}


data "coder_parameter" "home_volume_size" {
  name        = "How large would you like your home volume to be (in GB)?"
  description = "This volume will be mounted to /home/coder."
  type        = "number"
  default     = "20"
  mutable     = false
  validation {
    min = 1
    max = 999999
  }
}

data "coder_parameter" "region" {
  name        = "Which region would you like to use?"
  description = "This is the region where your workspace will be created."
  icon        = "/emojis/1f30e.png"
  type        = "string"
  default     = "ams3"
  mutable     = false
  option {
    name  = "New York 1"
    value = "nyc1"
    icon  = "/emojis/1f1fa_1f1f8.png"
  }
  option {
    name  = "New York 2"
    value = "nyc2"
    icon  = "/emojis/1f1fa_1f1f8.png"
  }
  option {
    name  = "New York 3"
    value = "nyc3"
    icon  = "/emojis/1f1fa_1f1f8.png"
  }
  option {
    name  = "San Francisco 1"
    value = "sfo1"
    icon  = "/emojis/1f1fa_1f1f8.png"
  }
  option {
    name  = "San Francisco 2"
    value = "sfo2"
    icon  = "/emojis/1f1fa_1f1f8.png"
  }
  option {
    name  = "San Francisco 3"
    value = "sfo3"
    icon  = "/emojis/1f1fa_1f1f8.png"
  }
  option {
    name  = "Amsterdam 2"
    value = "ams2"
    icon  = "/emojis/1f1f3_1f1f1.png"
  }
  option {
    name  = "Amsterdam 3"
    value = "ams3"
    icon  = "/emojis/1f1f3_1f1f1.png"
  }
  option {
    name  = "Singapore 1"
    value = "sgp1"
    icon  = "/emojis/1f1f8_1f1ec.png"
  }
  option {
    name  = "London 1"
    value = "lon1"
    icon  = "/emojis/1f1ec_1f1e7.png"
  }
  option {
    name  = "Frankfurt 1"
    value = "fra1"
    icon  = "/emojis/1f1e9_1f1ea.png"
  }
  option {
    name  = "Toronto 1"
    value = "tor1"
    icon  = "/emojis/1f1e8_1f1e6.png"
  }
  option {
    name  = "Bangalore 1"
    value = "blr1"
    icon  = "/emojis/1f1ee_1f1f3.png"
  }
}

# Configure the DigitalOcean Provider
provider "digitalocean" {
  # Recommended: use environment variable DIGITALOCEAN_TOKEN with your personal access token when starting coderd
  # alternatively, you can pass the token via a variable.
}

data "coder_workspace" "me" {}

resource "coder_agent" "main" {
  os   = "linux"
  arch = "amd64"

  login_before_ready = false
}

resource "digitalocean_volume" "home_volume" {
  region                   = data.coder_parameter.region.value
  name                     = "coder-${data.coder_workspace.me.id}-home"
  size                     = data.coder_parameter.home_volume_size.value
  initial_filesystem_type  = "ext4"
  initial_filesystem_label = "coder-home"
  # Protect the volume from being deleted due to changes in attributes.
  lifecycle {
    ignore_changes = all
  }
}

resource "digitalocean_droplet" "workspace" {
  region = data.coder_parameter.region.value
  count  = data.coder_workspace.me.start_count
  name   = "coder-${data.coder_workspace.me.owner}-${data.coder_workspace.me.name}"
  image  = data.coder_parameter.droplet_image.value
  size   = data.coder_parameter.droplet_size.value

  volume_ids = [digitalocean_volume.home_volume.id]
  user_data = templatefile("cloud-config.yaml.tftpl", {
    username          = data.coder_workspace.me.owner
    home_volume_label = digitalocean_volume.home_volume.initial_filesystem_label
    init_script       = base64encode(coder_agent.main.init_script)
    coder_agent_token = coder_agent.main.token
  })
  # Required to provision Fedora.
  ssh_keys = data.coder_parameter.step1_do_project_id.value > 0 ? [data.coder_parameter.step2_do_admin_ssh_key.value] : []
}

resource "digitalocean_project_resources" "project" {
  project = data.coder_parameter.step1_do_project_id.value
  # Workaround for terraform plan when using count.
  resources = length(digitalocean_droplet.workspace) > 0 ? [
    digitalocean_volume.home_volume.urn,
    digitalocean_droplet.workspace[0].urn
    ] : [
    digitalocean_volume.home_volume.urn
  ]
}

resource "coder_metadata" "workspace-info" {
  count       = data.coder_workspace.me.start_count
  resource_id = digitalocean_droplet.workspace[0].id

  item {
    key   = "region"
    value = digitalocean_droplet.workspace[0].region
  }
  item {
    key   = "image"
    value = digitalocean_droplet.workspace[0].image
  }
}

resource "coder_metadata" "volume-info" {
  resource_id = digitalocean_volume.home_volume.id

  item {
    key   = "size"
    value = "${digitalocean_volume.home_volume.size} GiB"
  }
}
