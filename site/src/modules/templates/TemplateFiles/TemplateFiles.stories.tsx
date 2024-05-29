import type { Meta, StoryObj } from "@storybook/react";
import { chromatic } from "testHelpers/chromatic";
import { TemplateFiles } from "./TemplateFiles";

const exampleFiles = {
  "README.md":
    "---\nname: Develop in Docker\ndescription: Develop inside Docker containers using your local daemon\ntags: [local, docker]\nicon: /icon/docker.png\n---\n\n# docker\n\nTo get started, run `coder templates init`. When prompted, select this template.\nFollow the on-screen instructions to proceed.\n\n## Editing the image\n\nEdit the `Dockerfile` and run `coder templates push` to update workspaces.\n\n## code-server\n\n`code-server` is installed via the `startup_script` argument in the `coder_agent`\nresource block. The `coder_app` resource is defined to access `code-server` through\nthe dashboard UI over `localhost:13337`.\n\n## Extending this template\n\nSee the [kreuzwerker/docker](https://registry.terraform.io/providers/kreuzwerker/docker) Terraform provider documentation to\nadd the following features to your Coder template:\n\n- SSH/TCP docker host\n- Registry authentication\n- Build args\n- Volume mounts\n- Custom container spec\n- More\n\nWe also welcome contributions!\n",
  "build/Dockerfile":
    'FROM ubuntu\n\nRUN apt-get update \\\n\t&& apt-get install -y \\\n\tcurl \\\n\tgit \\\n\tgolang \\\n\tsudo \\\n\tvim \\\n\twget \\\n\t&& rm -rf /var/lib/apt/lists/*\n\nARG USER=coder\nRUN useradd --groups sudo --no-create-home --shell /bin/bash ${USER} \\\n\t&& echo "${USER} ALL=(ALL) NOPASSWD:ALL" >/etc/sudoers.d/${USER} \\\n\t&& chmod 0440 /etc/sudoers.d/${USER}\nUSER ${USER}\nWORKDIR /home/${USER}\n',
  "main.tf":
    'terraform {\n  required_providers {\n    coder = {\n      source = "coder/coder"\n    }\n    docker = {\n      source = "kreuzwerker/docker"\n    }\n  }\n}\n\nlocals {\n  username = data.coder_workspace_owner.me.name\n}\n\ndata "coder_provisioner" "me" {\n}\n\nprovider "docker" {\n}\n\ndata "coder_workspace" "me" {\n}\n\nresource "coder_agent" "main" {\n  arch                   = data.coder_provisioner.me.arch\n  os                     = "linux"\n  startup_script_timeout = 180\n  startup_script         = <<-EOT\n    set -e\n\n    # install and start code-server\n    curl -fsSL https://code-server.dev/install.sh | sh -s -- --method=standalone --prefix=/tmp/code-server --version 4.11.0\n    /tmp/code-server/bin/code-server --auth none --port 13337 >/tmp/code-server.log 2>&1 &\n  EOT\n\n  # These environment variables allow you to make Git commits right away after creating a\n  # workspace. Note that they take precedence over configuration defined in ~/.gitconfig!\n  # You can remove this block if you\'d prefer to configure Git manually or using\n  # dotfiles. (see docs/dotfiles.md)\n  env = {\n    GIT_AUTHOR_NAME     = "${data.coder_workspace_owner.me.name}"\n    GIT_COMMITTER_NAME  = "${data.coder_workspace_owner.me.name}"\n    GIT_AUTHOR_EMAIL    = "${data.coder_workspace_owner.me.email}"\n    GIT_COMMITTER_EMAIL = "${data.coder_workspace_owner.me.email}"\n  }\n\n  # The following metadata blocks are optional. They are used to display\n  # information about your workspace in the dashboard. You can remove them\n  # if you don\'t want to display any information.\n  # For basic resources, you can use the `coder stat` command.\n  # If you need more control, you can write your own script.\n  metadata {\n    display_name = "CPU Usage"\n    key          = "0_cpu_usage"\n    script       = "coder stat cpu"\n    interval     = 10\n    timeout      = 1\n  }\n\n  metadata {\n    display_name = "RAM Usage"\n    key          = "1_ram_usage"\n    script       = "coder stat mem"\n    interval     = 10\n    timeout      = 1\n  }\n\n  metadata {\n    display_name = "Home Disk"\n    key          = "3_home_disk"\n    script       = "coder stat disk --path $${HOME}"\n    interval     = 60\n    timeout      = 1\n  }\n\n  metadata {\n    display_name = "CPU Usage (Host)"\n    key          = "4_cpu_usage_host"\n    script       = "coder stat cpu --host"\n    interval     = 10\n    timeout      = 1\n  }\n\n  metadata {\n    display_name = "Memory Usage (Host)"\n    key          = "5_mem_usage_host"\n    script       = "coder stat mem --host"\n    interval     = 10\n    timeout      = 1\n  }\n\n  metadata {\n    display_name = "Load Average (Host)"\n    key          = "6_load_host"\n    # get load avg scaled by number of cores\n    script   = <<EOT\n      echo "`cat /proc/loadavg | awk \'{ print $1 }\'` `nproc`" | awk \'{ printf "%0.2f", $1/$2 }\'\n    EOT\n    interval = 60\n    timeout  = 1\n  }\n\n  metadata {\n    display_name = "Swap Usage (Host)"\n    key          = "7_swap_host"\n    script       = <<EOT\n      free -b | awk \'/^Swap/ { printf("%.1f/%.1f", $3/1024.0/1024.0/1024.0, $2/1024.0/1024.0/1024.0) }\'\n    EOT\n    interval     = 10\n    timeout      = 1\n  }\n}\n\nresource "coder_app" "code-server" {\n  agent_id     = coder_agent.main.id\n  slug         = "code-server"\n  display_name = "code-server"\n  url          = "http://localhost:13337/?folder=/home/${local.username}"\n  icon         = "/icon/code.svg"\n  subdomain    = false\n  share        = "owner"\n\n  healthcheck {\n    url       = "http://localhost:13337/healthz"\n    interval  = 5\n    threshold = 6\n  }\n}\n\nresource "docker_volume" "home_volume" {\n  name = "coder-${data.coder_workspace.me.id}-home"\n  # Protect the volume from being deleted due to changes in attributes.\n  lifecycle {\n    ignore_changes = all\n  }\n  # Add labels in Docker to keep track of orphan resources.\n  labels {\n    label = "coder.owner"\n    value = data.coder_workspace_owner.me.name\n  }\n  labels {\n    label = "coder.owner_id"\n    value = data.coder_workspace_owner.me.id\n  }\n  labels {\n    label = "coder.workspace_id"\n    value = data.coder_workspace.me.id\n  }\n  # This field becomes outdated if the workspace is renamed but can\n  # be useful for debugging or cleaning out dangling volumes.\n  labels {\n    label = "coder.workspace_name_at_creation"\n    value = data.coder_workspace.me.name\n  }\n}\n\nresource "docker_image" "main" {\n  name = "coder-${data.coder_workspace.me.id}"\n  build {\n    context = "./build"\n    build_args = {\n      USER = local.username\n    }\n  }\n  triggers = {\n    dir_sha1 = sha1(join("", [for f in fileset(path.module, "build/*") : filesha1(f)]))\n  }\n}\n\nresource "docker_container" "workspace" {\n  count = data.coder_workspace.me.start_count\n  image = docker_image.main.name\n  # Uses lower() to avoid Docker restriction on container names.\n  name = "coder-${data.coder_workspace_owner.me.name}-${lower(data.coder_workspace.me.name)}"\n  # Hostname makes the shell more user friendly: coder@my-workspace:~$\n  hostname = data.coder_workspace.me.name\n  # Use the docker gateway if the access URL is 127.0.0.1\n  entrypoint = ["sh", "-c", replace(coder_agent.main.init_script, "/localhost|127\\\\.0\\\\.0\\\\.1/", "host.docker.internal")]\n  env        = ["CODER_AGENT_TOKEN=${coder_agent.main.token}"]\n  host {\n    host = "host.docker.internal"\n    ip   = "host-gateway"\n  }\n  volumes {\n    container_path = "/home/${local.username}"\n    volume_name    = docker_volume.home_volume.name\n    read_only      = false\n  }\n  # Add labels in Docker to keep track of orphan resources.\n  labels {\n    label = "coder.owner"\n    value = data.coder_workspace_owner.me.name\n  }\n  labels {\n    label = "coder.owner_id"\n    value = data.coder_workspace_owner.me.id\n  }\n  labels {\n    label = "coder.workspace_id"\n    value = data.coder_workspace.me.id\n  }\n  labels {\n    label = "coder.workspace_name"\n    value = data.coder_workspace.me.name\n  }\n}\n',
};

const meta: Meta<typeof TemplateFiles> = {
  title: "modules/templates/TemplateFiles",
  parameters: { chromatic },
  component: TemplateFiles,
  args: {
    currentFiles: exampleFiles,
    baseFiles: exampleFiles,
  },
};

export default meta;
type Story = StoryObj<typeof TemplateFiles>;

const Example: Story = {};

export const WithDiff: Story = {
  args: {
    currentFiles: {
      ...exampleFiles,
      "main.tf": `${exampleFiles["main.tf"]} - with changes`,
    },
    baseFiles: exampleFiles,
  },
};

export { Example as TemplateFiles };
