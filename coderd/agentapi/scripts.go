package agentapi

import (
	"context"

	"github.com/google/uuid"
	"golang.org/x/xerrors"

	agentproto "github.com/coder/coder/v2/agent/proto"
	"github.com/coder/coder/v2/coderd/database"
	"github.com/coder/coder/v2/coderd/database/dbauthz"
)

type ScriptsAPI struct {
	Database database.Store
}

func (s *ScriptsAPI) ScriptCompleted(ctx context.Context, req *agentproto.WorkspaceAgentScriptCompletedRequest) (*agentproto.WorkspaceAgentScriptCompletedResponse, error) {
	res := &agentproto.WorkspaceAgentScriptCompletedResponse{}

	scriptID, err := uuid.FromBytes(req.Timing.ScriptId)
	if err != nil {
		return nil, xerrors.Errorf("script id from bytes: %w", err)
	}

	var stage database.WorkspaceAgentScriptTimingStage
	switch req.Timing.Stage {
	case agentproto.Timing_START:
		stage = database.WorkspaceAgentScriptTimingStageStart
	case agentproto.Timing_STOP:
		stage = database.WorkspaceAgentScriptTimingStageStop
	case agentproto.Timing_CRON:
		stage = database.WorkspaceAgentScriptTimingStageCron
	}

	//nolint:gocritic // We need permissions to write to the DB here and we are in the context of the agent.
	ctx = dbauthz.AsProvisionerd(ctx)
	err = s.Database.InsertWorkspaceAgentScriptTimings(ctx, database.InsertWorkspaceAgentScriptTimingsParams{
		ScriptID:  scriptID,
		Stage:     stage,
		StartedAt: req.Timing.Start.AsTime(),
		EndedAt:   req.Timing.End.AsTime(),
		ExitCode:  req.Timing.ExitCode,
		TimedOut:  req.Timing.TimedOut,
	})
	if err != nil {
		return nil, xerrors.Errorf("insert workspace agent script timings into database: %w", err)
	}

	return res, nil
}
