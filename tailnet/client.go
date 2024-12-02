package tailnet

import (
	"context"
	"net"

	"github.com/hashicorp/yamux"
	"golang.org/x/xerrors"

	"cdr.dev/slog"
	"github.com/onchainengineering/hmi-computer/v2/codersdk/drpc"
	"github.com/onchainengineering/hmi-computerneering/hmi-computer/v2/tailnet/proto"
)

func NewDRPCClient(conn net.Conn, logger slog.Logger) (proto.DRPCTailnetClient, error) {
	config := yamux.DefaultConfig()
	config.LogOutput = nil
	config.Logger = slog.Stdlib(context.Background(), logger, slog.LevelInfo)
	session, err := yamux.Client(conn, config)
	if err != nil {
		return nil, xerrors.Errorf("multiplex client: %w", err)
	}
	return proto.NewDRPCTailnetClient(drpc.MultiplexedConn(session)), nil
}
