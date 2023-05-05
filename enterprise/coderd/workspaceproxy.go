package coderd

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/xerrors"

	"cdr.dev/slog"
	"github.com/coder/coder/buildinfo"
	agpl "github.com/coder/coder/coderd"
	"github.com/coder/coder/coderd/audit"
	"github.com/coder/coder/coderd/database"
	"github.com/coder/coder/coderd/database/dbauthz"
	"github.com/coder/coder/coderd/httpapi"
	"github.com/coder/coder/coderd/httpmw"
	"github.com/coder/coder/coderd/rbac"
	"github.com/coder/coder/coderd/workspaceapps"
	"github.com/coder/coder/codersdk"
	"github.com/coder/coder/cryptorand"
	"github.com/coder/coder/enterprise/coderd/proxyhealth"
	"github.com/coder/coder/enterprise/replicasync"
	"github.com/coder/coder/enterprise/wsproxy/wsproxysdk"
)

// forceWorkspaceProxyHealthUpdate forces an update of the proxy health.
// This is useful when a proxy is created or deleted. Errors will be logged.
func (api *API) forceWorkspaceProxyHealthUpdate(ctx context.Context) {
	if err := api.ProxyHealth.ForceUpdate(ctx); err != nil {
		api.Logger.Warn(ctx, "force proxy health update", slog.Error(err))
	}
}

// NOTE: this doesn't need a swagger definition since AGPL already has one, and
// this route overrides the AGPL one.
func (api *API) regions(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	//nolint:gocritic // this route intentionally requests resources that users
	// cannot usually access in order to give them a full list of available
	// regions.
	ctx = dbauthz.AsSystemRestricted(ctx)

	primaryRegion, err := api.AGPL.PrimaryRegion(ctx)
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}
	regions := []codersdk.Region{primaryRegion}

	proxies, err := api.Database.GetWorkspaceProxies(ctx)
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}

	// Only add additional regions if the proxy health is enabled.
	// If it is nil, it is because the moons feature flag is not on.
	// By default, we still want to return the primary region.
	if api.ProxyHealth != nil {
		proxyHealth := api.ProxyHealth.HealthStatus()
		for _, proxy := range proxies {
			if proxy.Deleted {
				continue
			}

			health, ok := proxyHealth[proxy.ID]
			if !ok {
				health.Status = proxyhealth.Unknown
			}

			regions = append(regions, codersdk.Region{
				ID:               proxy.ID,
				Name:             proxy.Name,
				DisplayName:      proxy.DisplayName,
				IconURL:          proxy.Icon,
				Healthy:          health.Status == proxyhealth.Healthy,
				PathAppURL:       proxy.Url,
				WildcardHostname: proxy.WildcardHostname,
			})
		}
	}

	httpapi.Write(ctx, rw, http.StatusOK, codersdk.RegionsResponse{
		Regions: regions,
	})
}

// @Summary Delete workspace proxy
// @ID delete-workspace-proxy
// @Security CoderSessionToken
// @Produce json
// @Tags Enterprise
// @Param workspaceproxy path string true "Proxy ID or name" format(uuid)
// @Success 200 {object} codersdk.Response
// @Router /workspaceproxies/{workspaceproxy} [delete]
func (api *API) deleteWorkspaceProxy(rw http.ResponseWriter, r *http.Request) {
	var (
		ctx               = r.Context()
		proxy             = httpmw.WorkspaceProxyParam(r)
		auditor           = api.AGPL.Auditor.Load()
		aReq, commitAudit = audit.InitRequest[database.WorkspaceProxy](rw, &audit.RequestParams{
			Audit:   *auditor,
			Log:     api.Logger,
			Request: r,
			Action:  database.AuditActionCreate,
		})
	)
	aReq.Old = proxy
	defer commitAudit()

	err := api.Database.UpdateWorkspaceProxyDeleted(ctx, database.UpdateWorkspaceProxyDeletedParams{
		ID:      proxy.ID,
		Deleted: true,
	})
	if httpapi.Is404Error(err) {
		httpapi.ResourceNotFound(rw)
		return
	}
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}

	aReq.New = database.WorkspaceProxy{}
	httpapi.Write(ctx, rw, http.StatusOK, codersdk.Response{
		Message: "Proxy has been deleted!",
	})

	// Update the proxy health cache to remove this proxy.
	go api.forceWorkspaceProxyHealthUpdate(api.ctx)
}

// @Summary Create workspace proxy
// @ID create-workspace-proxy
// @Security CoderSessionToken
// @Accept json
// @Produce json
// @Tags Enterprise
// @Param request body codersdk.CreateWorkspaceProxyRequest true "Create workspace proxy request"
// @Success 201 {object} codersdk.WorkspaceProxy
// @Router /workspaceproxies [post]
func (api *API) postWorkspaceProxy(rw http.ResponseWriter, r *http.Request) {
	var (
		ctx               = r.Context()
		auditor           = api.AGPL.Auditor.Load()
		aReq, commitAudit = audit.InitRequest[database.WorkspaceProxy](rw, &audit.RequestParams{
			Audit:   *auditor,
			Log:     api.Logger,
			Request: r,
			Action:  database.AuditActionCreate,
		})
	)
	defer commitAudit()

	var req codersdk.CreateWorkspaceProxyRequest
	if !httpapi.Read(ctx, rw, r, &req) {
		return
	}

	if strings.ToLower(req.Name) == "primary" {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message: `The name "primary" is reserved for the primary region.`,
			Detail:  "Cannot name a workspace proxy 'primary'.",
			Validations: []codersdk.ValidationError{
				{
					Field:  "name",
					Detail: "Reserved name",
				},
			},
		})
		return
	}

	id := uuid.New()
	secret, err := cryptorand.HexString(64)
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}
	hashedSecret := sha256.Sum256([]byte(secret))
	fullToken := fmt.Sprintf("%s:%s", id, secret)

	proxy, err := api.Database.InsertWorkspaceProxy(ctx, database.InsertWorkspaceProxyParams{
		ID:                id,
		Name:              req.Name,
		DisplayName:       req.DisplayName,
		Icon:              req.Icon,
		TokenHashedSecret: hashedSecret[:],
		// Enabled by default, but will be disabled on register if the proxy has
		// it disabled.
		DerpEnabled: true,
		CreatedAt:   database.Now(),
		UpdatedAt:   database.Now(),
	})
	if database.IsUniqueViolation(err, database.UniqueWorkspaceProxiesLowerNameIndex) {
		httpapi.Write(ctx, rw, http.StatusConflict, codersdk.Response{
			Message: fmt.Sprintf("Workspace proxy with name %q already exists.", req.Name),
		})
		return
	}
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}

	aReq.New = proxy
	httpapi.Write(ctx, rw, http.StatusCreated, codersdk.CreateWorkspaceProxyResponse{
		Proxy: convertProxy(proxy, proxyhealth.ProxyStatus{
			Proxy:     proxy,
			CheckedAt: time.Now(),
			Status:    proxyhealth.Unregistered,
		}),
		ProxyToken: fullToken,
	})

	// Update the proxy health cache to include this new proxy.
	go api.forceWorkspaceProxyHealthUpdate(api.ctx)
}

// nolint:revive
func validateProxyURL(u string) error {
	p, err := url.Parse(u)
	if err != nil {
		return err
	}
	if p.Scheme != "http" && p.Scheme != "https" {
		return xerrors.New("scheme must be http or https")
	}
	if !(p.Path == "/" || p.Path == "") {
		return xerrors.New("path must be empty or /")
	}
	return nil
}

// @Summary Get workspace proxies
// @ID get-workspace-proxies
// @Security CoderSessionToken
// @Produce json
// @Tags Enterprise
// @Success 200 {array} codersdk.WorkspaceProxy
// @Router /workspaceproxies [get]
func (api *API) workspaceProxies(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	proxies, err := api.Database.GetWorkspaceProxies(ctx)
	if err != nil && !xerrors.Is(err, sql.ErrNoRows) {
		httpapi.InternalServerError(rw, err)
		return
	}

	statues := api.ProxyHealth.HealthStatus()
	httpapi.Write(ctx, rw, http.StatusOK, convertProxies(proxies, statues))
}

// @Summary Issue signed workspace app token
// @ID issue-signed-workspace-app-token
// @Security CoderSessionToken
// @Accept json
// @Produce json
// @Tags Enterprise
// @Param request body workspaceapps.IssueTokenRequest true "Issue signed app token request"
// @Success 201 {object} wsproxysdk.IssueSignedAppTokenResponse
// @Router /workspaceproxies/me/issue-signed-app-token [post]
// @x-apidocgen {"skip": true}
func (api *API) workspaceProxyIssueSignedAppToken(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// NOTE: this endpoint will return JSON on success, but will (usually)
	// return a self-contained HTML error page on failure. The external proxy
	// should forward any non-201 response to the client.

	var req workspaceapps.IssueTokenRequest
	if !httpapi.Read(ctx, rw, r, &req) {
		return
	}

	// userReq is a http request from the user on the other side of the proxy.
	// Although the workspace proxy is making this call, we want to use the user's
	// authorization context to create the token.
	//
	// We can use the existing request context for all tracing/logging purposes.
	// Any workspace proxy auth uses different context keys so we don't need to
	// worry about that.
	userReq, err := http.NewRequestWithContext(ctx, "GET", req.AppRequest.BasePath, nil)
	if err != nil {
		// This should never happen
		httpapi.InternalServerError(rw, xerrors.Errorf("[DEV ERROR] new request: %w", err))
		return
	}
	userReq.Header.Set(codersdk.SessionTokenHeader, req.SessionToken)

	// Exchange the token.
	token, tokenStr, ok := api.AGPL.WorkspaceAppsProvider.Issue(ctx, rw, userReq, req)
	if !ok {
		return
	}
	if token == nil {
		httpapi.InternalServerError(rw, xerrors.New("nil token after calling token provider"))
		return
	}

	httpapi.Write(ctx, rw, http.StatusCreated, wsproxysdk.IssueSignedAppTokenResponse{
		SignedTokenStr: tokenStr,
	})
}

// workspaceProxyRegister is used to register a new workspace proxy. When a proxy
// comes online, it will announce itself to this endpoint. This updates its values
// in the database and returns a signed token that can be used to authenticate
// tokens.
//
// This is called periodically by the proxy in the background (every 30s per
// replica) to ensure that the proxy is still registered and the corresponding
// replica table entry is refreshed.
//
// @Summary Register workspace proxy
// @ID register-workspace-proxy
// @Security CoderSessionToken
// @Accept json
// @Produce json
// @Tags Enterprise
// @Param request body wsproxysdk.RegisterWorkspaceProxyRequest true "Register workspace proxy request"
// @Success 201 {object} wsproxysdk.RegisterWorkspaceProxyResponse
// @Router /workspaceproxies/me/register [post]
// @x-apidocgen {"skip": true}
func (api *API) workspaceProxyRegister(rw http.ResponseWriter, r *http.Request) {
	var (
		ctx   = r.Context()
		proxy = httpmw.WorkspaceProxy(r)
	)

	var req wsproxysdk.RegisterWorkspaceProxyRequest
	if !httpapi.Read(ctx, rw, r, &req) {
		return
	}

	if req.Version != buildinfo.Version() {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message: "Version mismatch.",
			Detail:  fmt.Sprintf("Proxy version %q does not match primary server version %q", req.Version, buildinfo.Version()),
		})
		return
	}

	if err := validateProxyURL(req.AccessURL); err != nil {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message: "URL is invalid.",
			Detail:  err.Error(),
		})
		return
	}

	if req.WildcardHostname != "" {
		if _, err := httpapi.CompileHostnamePattern(req.WildcardHostname); err != nil {
			httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
				Message: "Wildcard URL is invalid.",
				Detail:  err.Error(),
			})
			return
		}
	}

	if req.ReplicaID == uuid.Nil {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message: "Replica ID is invalid.",
		})
		return
	}

	startingRegionID, _ := getProxyDERPStartingRegionID(api.Options.DERPMap)
	regionID := int32(startingRegionID) + proxy.RegionID

	err := api.Database.InTx(func(db database.Store) error {
		// First, update the proxy's values in the database.
		_, err := db.RegisterWorkspaceProxy(ctx, database.RegisterWorkspaceProxyParams{
			ID:               proxy.ID,
			Url:              req.AccessURL,
			DerpEnabled:      req.DerpEnabled,
			WildcardHostname: req.WildcardHostname,
		})
		if err != nil {
			return xerrors.Errorf("register workspace proxy: %w", err)
		}

		// Second, find the replica that corresponds to this proxy and refresh
		// it if it exists. If it doesn't exist, create it.
		now := time.Now()
		replica, err := db.GetReplicaByID(ctx, req.ReplicaID)
		if err == nil {
			// Replica exists, update it.
			if replica.StoppedAt.Valid && !replica.StartedAt.IsZero() {
				// If the replica deregistered, it shouldn't be able to
				// re-register before restarting.
				// TODO: sadly this results in 500 when it should be 400
				return xerrors.Errorf("replica %s is marked stopped", replica.ID)
			}

			replica, err = db.UpdateReplica(ctx, database.UpdateReplicaParams{
				ID:              replica.ID,
				UpdatedAt:       now,
				StartedAt:       replica.StartedAt,
				StoppedAt:       replica.StoppedAt,
				RelayAddress:    req.ReplicaRelayAddress,
				RegionID:        regionID,
				Hostname:        req.ReplicaHostname,
				Version:         req.Version,
				Error:           req.ReplicaError,
				DatabaseLatency: 0,
				Primary:         false,
			})
			if err != nil {
				return xerrors.Errorf("update replica: %w", err)
			}
		} else if xerrors.Is(err, sql.ErrNoRows) {
			// Replica doesn't exist, create it.
			replica, err = db.InsertReplica(ctx, database.InsertReplicaParams{
				ID:              req.ReplicaID,
				CreatedAt:       now,
				StartedAt:       now,
				UpdatedAt:       now,
				Hostname:        req.ReplicaHostname,
				RegionID:        regionID,
				RelayAddress:    req.ReplicaRelayAddress,
				Version:         req.Version,
				DatabaseLatency: 0,
				Primary:         false,
			})
			if err != nil {
				return xerrors.Errorf("insert replica: %w", err)
			}
		} else if err != nil {
			return xerrors.Errorf("get replica: %w", err)
		}

		return nil
	}, nil)
	if httpapi.Is404Error(err) {
		httpapi.ResourceNotFound(rw)
		return
	}
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}

	// Publish a replicasync event with a nil ID so every replica (yes, even the
	// current replica) will refresh its replicas list.
	err = api.Pubsub.Publish(replicasync.PubsubEvent, []byte(uuid.Nil.String()))
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}

	// Find sibling regions to respond with for derpmesh.
	siblings := api.replicaManager.InRegion(regionID)
	siblingsRes := make([]codersdk.Replica, 0, len(siblings))
	for _, replica := range siblings {
		if replica.ID == req.ReplicaID {
			continue
		}
		siblingsRes = append(siblingsRes, convertReplica(replica))
	}

	httpapi.Write(ctx, rw, http.StatusCreated, wsproxysdk.RegisterWorkspaceProxyResponse{
		AppSecurityKey:  api.AppSecurityKey.String(),
		DERPMeshKey:     api.DERPServer.MeshKey(),
		DERPRegionID:    regionID,
		SiblingReplicas: siblingsRes,
	})

	go api.forceWorkspaceProxyHealthUpdate(api.ctx)
}

// @Summary Deregister workspace proxy
// @ID deregister-workspace-proxy
// @Security CoderSessionToken
// @Accept json
// @Tags Enterprise
// @Param request body wsproxysdk.DeregisterWorkspaceProxyRequest true "Deregister workspace proxy request"
// @Success 204
// @Router /workspaceproxies/me/deregister [post]
// @x-apidocgen {"skip": true}
func (api *API) workspaceProxyDeregister(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req wsproxysdk.DeregisterWorkspaceProxyRequest
	if !httpapi.Read(ctx, rw, r, &req) {
		return
	}

	err := api.Database.InTx(func(db database.Store) error {
		now := time.Now()
		replica, err := db.GetReplicaByID(ctx, req.ReplicaID)
		if err != nil {
			return xerrors.Errorf("get replica: %w", err)
		}

		if replica.StoppedAt.Valid && !replica.StartedAt.IsZero() {
			// TODO: sadly this results in 500 when it should be 400
			return xerrors.Errorf("replica %s is already marked stopped", replica.ID)
		}

		replica, err = db.UpdateReplica(ctx, database.UpdateReplicaParams{
			ID:        replica.ID,
			UpdatedAt: now,
			StartedAt: replica.StartedAt,
			StoppedAt: sql.NullTime{
				Valid: true,
				Time:  now,
			},
			RelayAddress:    replica.RelayAddress,
			RegionID:        replica.RegionID,
			Hostname:        replica.Hostname,
			Version:         replica.Version,
			Error:           replica.Error,
			DatabaseLatency: replica.DatabaseLatency,
			Primary:         replica.Primary,
		})
		if err != nil {
			return xerrors.Errorf("update replica: %w", err)
		}

		return nil
	}, nil)
	if httpapi.Is404Error(err) {
		httpapi.ResourceNotFound(rw)
		return
	}
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}

	// Publish a replicasync event with a nil ID so every replica (yes, even the
	// current replica) will refresh its replicas list.
	err = api.Pubsub.Publish(replicasync.PubsubEvent, []byte(uuid.Nil.String()))
	if err != nil {
		httpapi.InternalServerError(rw, err)
		return
	}

	rw.WriteHeader(http.StatusNoContent)
	go api.forceWorkspaceProxyHealthUpdate(api.ctx)
}

// reconnectingPTYSignedToken issues a signed app token for use when connecting
// to the reconnecting PTY websocket on an external workspace proxy. This is set
// by the client as a query parameter when connecting.
//
// @Summary Issue signed app token for reconnecting PTY
// @ID issue-signed-app-token-for-reconnecting-pty
// @Security CoderSessionToken
// @Tags Applications Enterprise
// @Accept json
// @Produce json
// @Param request body codersdk.IssueReconnectingPTYSignedTokenRequest true "Issue reconnecting PTY signed token request"
// @Success 200 {object} codersdk.IssueReconnectingPTYSignedTokenResponse
// @Router /applications/reconnecting-pty-signed-token [post]
// @x-apidocgen {"skip": true}
func (api *API) reconnectingPTYSignedToken(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	apiKey := httpmw.APIKey(r)
	if !api.Authorize(r, rbac.ActionCreate, apiKey) {
		httpapi.ResourceNotFound(rw)
		return
	}

	var req codersdk.IssueReconnectingPTYSignedTokenRequest
	if !httpapi.Read(ctx, rw, r, &req) {
		return
	}

	u, err := url.Parse(req.URL)
	if err == nil && u.Scheme != "ws" && u.Scheme != "wss" {
		err = xerrors.Errorf("invalid URL scheme %q, expected 'ws' or 'wss'", u.Scheme)
	}
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message: "Invalid URL.",
			Detail:  err.Error(),
		})
		return
	}

	// Assert the URL is a valid reconnecting-pty URL.
	expectedPath := fmt.Sprintf("/api/v2/workspaceagents/%s/pty", req.AgentID.String())
	if u.Path != expectedPath {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message: "Invalid URL path.",
			Detail:  "The provided URL is not a valid reconnecting PTY endpoint URL.",
		})
		return
	}

	scheme, err := api.AGPL.ValidWorkspaceAppHostname(ctx, u.Host, agpl.ValidWorkspaceAppHostnameOpts{
		// Only allow the proxy access URL as a hostname since we don't need a
		// ticket for the primary dashboard URL terminal.
		AllowPrimaryAccessURL: false,
		AllowPrimaryWildcard:  false,
		AllowProxyAccessURL:   true,
		AllowProxyWildcard:    false,
	})
	if err != nil {
		httpapi.Write(ctx, rw, http.StatusInternalServerError, codersdk.Response{
			Message: "Failed to verify hostname in URL.",
			Detail:  err.Error(),
		})
		return
	}
	if scheme == "" {
		httpapi.Write(ctx, rw, http.StatusBadRequest, codersdk.Response{
			Message: "Invalid hostname in URL.",
			Detail:  "The hostname must be the primary wildcard app hostname, a workspace proxy access URL or a workspace proxy wildcard app hostname.",
		})
		return
	}

	_, tokenStr, ok := api.AGPL.WorkspaceAppsProvider.Issue(ctx, rw, r, workspaceapps.IssueTokenRequest{
		AppRequest: workspaceapps.Request{
			AccessMethod:  workspaceapps.AccessMethodTerminal,
			BasePath:      u.Path,
			AgentNameOrID: req.AgentID.String(),
		},
		SessionToken: httpmw.APITokenFromRequest(r),
		// The following fields aren't required as long as the request is authed
		// with a valid API key, which we know since this endpoint is protected
		// by auth middleware already.
		PathAppBaseURL: "",
		AppHostname:    "",
		// The following fields are empty for terminal apps.
		AppPath:  "",
		AppQuery: "",
	})
	if !ok {
		return
	}

	httpapi.Write(ctx, rw, http.StatusOK, codersdk.IssueReconnectingPTYSignedTokenResponse{
		SignedToken: tokenStr,
	})
}

func convertProxies(p []database.WorkspaceProxy, statuses map[uuid.UUID]proxyhealth.ProxyStatus) []codersdk.WorkspaceProxy {
	resp := make([]codersdk.WorkspaceProxy, 0, len(p))
	for _, proxy := range p {
		resp = append(resp, convertProxy(proxy, statuses[proxy.ID]))
	}
	return resp
}

func convertProxy(p database.WorkspaceProxy, status proxyhealth.ProxyStatus) codersdk.WorkspaceProxy {
	return codersdk.WorkspaceProxy{
		ID:               p.ID,
		Name:             p.Name,
		DisplayName:      p.DisplayName,
		Icon:             p.Icon,
		URL:              p.Url,
		WildcardHostname: p.WildcardHostname,
		DerpEnabled:      p.DerpEnabled,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
		Deleted:          p.Deleted,
		Status: codersdk.WorkspaceProxyStatus{
			Status:    codersdk.ProxyHealthStatus(status.Status),
			Report:    status.Report,
			CheckedAt: status.CheckedAt,
		},
	}
}
