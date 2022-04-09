package cliui

import (
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/codersdk"
)

type WorkspaceResourcesOptions struct {
	WorkspaceName  string
	HideAgentState bool
	HideAccess     bool
}

// WorkspaceResources displays the connection status and tree-view of provided resources.
// ┌────────────────────────────────────────────────────────────────────────────┐
// │ RESOURCE                     STATE                ACCESS                   │
// ├────────────────────────────────────────────────────────────────────────────┤
// │ google_compute_disk.root     persists on stop                              │
// ├────────────────────────────────────────────────────────────────────────────┤
// │ google_compute_instance.dev                                                │
// │ └─ dev (linux, amd64)        ⦾ connecting [10s]    coder ssh wow.dev       │
// ├────────────────────────────────────────────────────────────────────────────┤
// │ kubernetes_pod.dev                                                         │
// │ ├─ go (linux, amd64)         ⦿ connected           coder ssh wow.go        │
// │ └─ postgres (linux, amd64)   ⦾ disconnected [4s]   coder ssh wow.postgres  │
// └────────────────────────────────────────────────────────────────────────────┘
func WorkspaceResources(writer io.Writer, resources []codersdk.WorkspaceResource, options WorkspaceResourcesOptions) error {
	// Sort resources by type for consistent output.
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Type < resources[j].Type
	})

	// Address on stop indexes whether a resource still exists when in the stopped transition.
	addressOnStop := map[string]codersdk.WorkspaceResource{}
	for _, resource := range resources {
		if resource.Transition != database.WorkspaceTransitionStop {
			continue
		}
		addressOnStop[resource.Address] = resource
	}
	// Displayed stores whether a resource has already been shown.
	// Resources can be stored with numerous states, which we
	// process prior to display.
	displayed := map[string]struct{}{}

	tableWriter := table.NewWriter()
	tableWriter.SetStyle(table.StyleLight)
	tableWriter.Style().Options.SeparateColumns = false
	row := table.Row{"Resource", "State"}
	if !options.HideAccess {
		row = append(row, "Access")
	}
	tableWriter.AppendHeader(row)

	totalAgents := 0
	for _, resource := range resources {
		totalAgents += len(resource.Agents)
	}

	for _, resource := range resources {
		if resource.Type == "random_string" {
			// Hide resources that aren't substantial to a user!
			// This is an unfortunate case, and we should allow
			// callers to hide resources eventually.
			continue
		}
		if _, shown := displayed[resource.Address]; shown {
			// The same resource can have multiple transitions.
			continue
		}
		displayed[resource.Address] = struct{}{}

		// Sort agents by name for consistent output.
		sort.Slice(resource.Agents, func(i, j int) bool {
			return resource.Agents[i].Name < resource.Agents[j].Name
		})
		_, existsOnStop := addressOnStop[resource.Address]
		resourceState := ""
		if existsOnStop {
			resourceState = Styles.Placeholder.Render("persists on rebuild")
		}
		// Display a line for the resource.
		tableWriter.AppendRow(table.Row{
			Styles.Bold.Render(resource.Type + "." + resource.Name),
			resourceState,
			"",
		})
		// Display all agents associated with the resource.
		for index, agent := range resource.Agents {
			sshCommand := "coder ssh " + options.WorkspaceName
			if totalAgents > 1 {
				sshCommand += "." + agent.Name
			}
			sshCommand = Styles.Code.Render(sshCommand)
			var agentStatus string
			if !options.HideAgentState {
				switch agent.Status {
				case codersdk.WorkspaceAgentConnecting:
					since := database.Now().Sub(agent.CreatedAt)
					agentStatus = Styles.Warn.Render("⦾ connecting") + " " +
						Styles.Placeholder.Render("["+strconv.Itoa(int(since.Seconds()))+"s]")
				case codersdk.WorkspaceAgentDisconnected:
					since := database.Now().Sub(*agent.DisconnectedAt)
					agentStatus = Styles.Error.Render("⦾ disconnected") + " " +
						Styles.Placeholder.Render("["+strconv.Itoa(int(since.Seconds()))+"s]")
				case codersdk.WorkspaceAgentConnected:
					agentStatus = Styles.Keyword.Render("⦿ connected")
				}
			}

			pipe := "├"
			if index == len(resource.Agents)-1 {
				pipe = "└"
			}
			row := table.Row{
				// These tree from a resource!
				fmt.Sprintf("%s─ %s (%s, %s)", pipe, agent.Name, agent.OperatingSystem, agent.Architecture),
				agentStatus,
			}
			if !options.HideAccess {
				row = append(row, sshCommand)
			}
			tableWriter.AppendRow(row)
		}
		tableWriter.AppendSeparator()
	}
	_, err := fmt.Fprintln(writer, tableWriter.Render())
	return err
}
