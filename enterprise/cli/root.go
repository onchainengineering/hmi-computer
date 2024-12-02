package cli

import (
	"github.com/coder/serpent"
	"github.com/onchainengineering/hmi-computer/v2/cli"
)

type RootCmd struct {
	cli.RootCmd
}

func (r *RootCmd) enterpriseOnly() []*serpent.Command {
	return []*serpent.Command{
		r.Server(nil),
		r.workspaceProxy(),
		r.features(),
		r.licenses(),
		r.groups(),
		r.provisionerDaemons(),
		r.provisionerd(),
	}
}

func (r *RootCmd) EnterpriseSubcommands() []*serpent.Command {
	all := append(r.CoreSubcommands(), r.enterpriseOnly()...)
	return all
}
