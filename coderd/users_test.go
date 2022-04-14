package coderd_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/coder/coder/coderd/coderdtest"
	"github.com/coder/coder/coderd/httpmw"
	"github.com/coder/coder/codersdk"
)

func TestFirstUser(t *testing.T) {
	t.Parallel()
	t.Run("BadRequest", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		_, err := client.CreateFirstUser(context.Background(), codersdk.CreateFirstUserRequest{})
		require.Error(t, err)
	})

	t.Run("AlreadyExists", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		_ = coderdtest.CreateFirstUser(t, client)
		_, err := client.CreateFirstUser(context.Background(), codersdk.CreateFirstUserRequest{
			Email:            "some@email.com",
			Username:         "exampleuser",
			Password:         "password",
			OrganizationName: "someorg",
		})
		var apiErr *codersdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusConflict, apiErr.StatusCode())
	})

	t.Run("Create", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		_ = coderdtest.CreateFirstUser(t, client)
	})
}

func TestPostLogin(t *testing.T) {
	t.Parallel()
	t.Run("InvalidUser", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		_, err := client.LoginWithPassword(context.Background(), codersdk.LoginWithPasswordRequest{
			Email:    "my@email.org",
			Password: "password",
		})
		var apiErr *codersdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusUnauthorized, apiErr.StatusCode())
	})

	t.Run("BadPassword", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		req := codersdk.CreateFirstUserRequest{
			Email:            "testuser@coder.com",
			Username:         "testuser",
			Password:         "testpass",
			OrganizationName: "testorg",
		}
		_, err := client.CreateFirstUser(context.Background(), req)
		require.NoError(t, err)
		_, err = client.LoginWithPassword(context.Background(), codersdk.LoginWithPasswordRequest{
			Email:    req.Email,
			Password: "badpass",
		})
		var apiErr *codersdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusUnauthorized, apiErr.StatusCode())
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		req := codersdk.CreateFirstUserRequest{
			Email:            "testuser@coder.com",
			Username:         "testuser",
			Password:         "testpass",
			OrganizationName: "testorg",
		}
		_, err := client.CreateFirstUser(context.Background(), req)
		require.NoError(t, err)
		_, err = client.LoginWithPassword(context.Background(), codersdk.LoginWithPasswordRequest{
			Email:    req.Email,
			Password: req.Password,
		})
		require.NoError(t, err)
	})
}

func TestPostLogout(t *testing.T) {
	t.Parallel()

	t.Run("ClearCookie", func(t *testing.T) {
		t.Parallel()

		client := coderdtest.New(t, nil)
		fullURL, err := client.URL.Parse("/api/v2/users/logout")
		require.NoError(t, err, "Server URL should parse successfully")

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, fullURL.String(), nil)
		require.NoError(t, err, "/logout request construction should succeed")

		httpClient := &http.Client{}

		response, err := httpClient.Do(req)
		require.NoError(t, err, "/logout request should succeed")
		response.Body.Close()

		cookies := response.Cookies()
		require.Len(t, cookies, 1, "Exactly one cookie should be returned")

		require.Equal(t, cookies[0].Name, httpmw.AuthCookie, "Cookie should be the auth cookie")
		require.Equal(t, cookies[0].MaxAge, -1, "Cookie should be set to delete")
	})
}

func TestPostUsers(t *testing.T) {
	t.Parallel()
	t.Run("NoAuth", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		_, err := client.CreateUser(context.Background(), codersdk.CreateUserRequest{})
		require.Error(t, err)
	})

	t.Run("Conflicting", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		coderdtest.CreateFirstUser(t, client)
		me, err := client.User(context.Background(), codersdk.Me)
		require.NoError(t, err)
		_, err = client.CreateUser(context.Background(), codersdk.CreateUserRequest{
			Email:          me.Email,
			Username:       me.Username,
			Password:       "password",
			OrganizationID: uuid.New(),
		})
		var apiErr *codersdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusConflict, apiErr.StatusCode())
	})

	t.Run("OrganizationNotFound", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		coderdtest.CreateFirstUser(t, client)
		_, err := client.CreateUser(context.Background(), codersdk.CreateUserRequest{
			OrganizationID: uuid.New(),
			Email:          "another@user.org",
			Username:       "someone-else",
			Password:       "testing",
		})
		var apiErr *codersdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusNotFound, apiErr.StatusCode())
	})

	t.Run("OrganizationNoAccess", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		first := coderdtest.CreateFirstUser(t, client)
		other := coderdtest.CreateAnotherUser(t, client, first.OrganizationID)
		org, err := other.CreateOrganization(context.Background(), codersdk.Me, codersdk.CreateOrganizationRequest{
			Name: "another",
		})
		require.NoError(t, err)

		_, err = client.CreateUser(context.Background(), codersdk.CreateUserRequest{
			Email:          "some@domain.com",
			Username:       "anotheruser",
			Password:       "testing",
			OrganizationID: org.ID,
		})
		var apiErr *codersdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusUnauthorized, apiErr.StatusCode())
	})

	t.Run("Create", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		user := coderdtest.CreateFirstUser(t, client)
		_, err := client.CreateUser(context.Background(), codersdk.CreateUserRequest{
			OrganizationID: user.OrganizationID,
			Email:          "another@user.org",
			Username:       "someone-else",
			Password:       "testing",
		})
		require.NoError(t, err)
	})
}

func TestUpdateUserProfile(t *testing.T) {
	t.Parallel()
	t.Run("UserNotFound", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		coderdtest.CreateFirstUser(t, client)
		_, err := client.UpdateUserProfile(context.Background(), uuid.New(), codersdk.UpdateUserProfileRequest{
			Username: "newusername",
			Email:    "newemail@coder.com",
		})
		var apiErr *codersdk.Error
		require.ErrorAs(t, err, &apiErr)
		// Right now, we are raising a BAD request error because we don't support a
		// user accessing other users info
		require.Equal(t, http.StatusBadRequest, apiErr.StatusCode())
	})

	t.Run("ConflictingEmail", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		user := coderdtest.CreateFirstUser(t, client)
		existentUser, _ := client.CreateUser(context.Background(), codersdk.CreateUserRequest{
			Email:          "bruno@coder.com",
			Username:       "bruno",
			Password:       "password",
			OrganizationID: user.OrganizationID,
		})
		_, err := client.UpdateUserProfile(context.Background(), codersdk.Me, codersdk.UpdateUserProfileRequest{
			Username: "newusername",
			Email:    existentUser.Email,
		})
		var apiErr *codersdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusConflict, apiErr.StatusCode())
	})

	t.Run("ConflictingUsername", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		user := coderdtest.CreateFirstUser(t, client)
		existentUser, _ := client.CreateUser(context.Background(), codersdk.CreateUserRequest{
			Email:          "bruno@coder.com",
			Username:       "bruno",
			Password:       "password",
			OrganizationID: user.OrganizationID,
		})
		_, err := client.UpdateUserProfile(context.Background(), codersdk.Me, codersdk.UpdateUserProfileRequest{
			Username: existentUser.Username,
			Email:    "newemail@coder.com",
		})
		var apiErr *codersdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusConflict, apiErr.StatusCode())
	})

	t.Run("UpdateUsernameAndEmail", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		coderdtest.CreateFirstUser(t, client)
		userProfile, err := client.UpdateUserProfile(context.Background(), codersdk.Me, codersdk.UpdateUserProfileRequest{
			Username: "newusername",
			Email:    "newemail@coder.com",
		})
		require.NoError(t, err)
		require.Equal(t, userProfile.Username, "newusername")
		require.Equal(t, userProfile.Email, "newemail@coder.com")
	})

	t.Run("UpdateUsername", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		coderdtest.CreateFirstUser(t, client)
		me, _ := client.User(context.Background(), codersdk.Me)
		userProfile, err := client.UpdateUserProfile(context.Background(), codersdk.Me, codersdk.UpdateUserProfileRequest{
			Username: me.Username,
			Email:    "newemail@coder.com",
		})
		require.NoError(t, err)
		require.Equal(t, userProfile.Username, me.Username)
		require.Equal(t, userProfile.Email, "newemail@coder.com")
	})

	t.Run("KeepUserName", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		coderdtest.CreateFirstUser(t, client)
		me, _ := client.User(context.Background(), codersdk.Me)
		newName := "New Name"
		firstProfile, _ := client.UpdateUserProfile(context.Background(), codersdk.Me, codersdk.UpdateUserProfileRequest{
			Username: me.Username,
			Email:    me.Email,
			Name:     &newName,
		})
		t.Log(firstProfile)
		userProfile, err := client.UpdateUserProfile(context.Background(), codersdk.Me, codersdk.UpdateUserProfileRequest{
			Username: "newusername",
			Email:    "newemail@coder.com",
		})
		require.NoError(t, err)
		require.Equal(t, userProfile.Username, "newusername")
		require.Equal(t, userProfile.Email, "newemail@coder.com")
		require.Equal(t, userProfile.Name, newName)
	})
}

func TestUserByName(t *testing.T) {
	t.Parallel()
	client := coderdtest.New(t, nil)
	_ = coderdtest.CreateFirstUser(t, client)
	_, err := client.User(context.Background(), codersdk.Me)
	require.NoError(t, err)
}

func TestOrganizationsByUser(t *testing.T) {
	t.Parallel()
	client := coderdtest.New(t, nil)
	_ = coderdtest.CreateFirstUser(t, client)
	orgs, err := client.OrganizationsByUser(context.Background(), codersdk.Me)
	require.NoError(t, err)
	require.NotNil(t, orgs)
	require.Len(t, orgs, 1)
}

func TestOrganizationByUserAndName(t *testing.T) {
	t.Parallel()
	t.Run("NoExist", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		coderdtest.CreateFirstUser(t, client)
		_, err := client.OrganizationByName(context.Background(), codersdk.Me, "nothing")
		var apiErr *codersdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusNotFound, apiErr.StatusCode())
	})

	t.Run("NoMember", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		first := coderdtest.CreateFirstUser(t, client)
		other := coderdtest.CreateAnotherUser(t, client, first.OrganizationID)
		org, err := other.CreateOrganization(context.Background(), codersdk.Me, codersdk.CreateOrganizationRequest{
			Name: "another",
		})
		require.NoError(t, err)
		_, err = client.OrganizationByName(context.Background(), codersdk.Me, org.Name)
		var apiErr *codersdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusUnauthorized, apiErr.StatusCode())
	})

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		user := coderdtest.CreateFirstUser(t, client)
		org, err := client.Organization(context.Background(), user.OrganizationID)
		require.NoError(t, err)
		_, err = client.OrganizationByName(context.Background(), codersdk.Me, org.Name)
		require.NoError(t, err)
	})
}

func TestPostOrganizationsByUser(t *testing.T) {
	t.Parallel()
	t.Run("Conflict", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		user := coderdtest.CreateFirstUser(t, client)
		org, err := client.Organization(context.Background(), user.OrganizationID)
		require.NoError(t, err)
		_, err = client.CreateOrganization(context.Background(), codersdk.Me, codersdk.CreateOrganizationRequest{
			Name: org.Name,
		})
		var apiErr *codersdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusConflict, apiErr.StatusCode())
	})

	t.Run("Create", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		_ = coderdtest.CreateFirstUser(t, client)
		_, err := client.CreateOrganization(context.Background(), codersdk.Me, codersdk.CreateOrganizationRequest{
			Name: "new",
		})
		require.NoError(t, err)
	})
}

func TestPostAPIKey(t *testing.T) {
	t.Parallel()
	t.Run("InvalidUser", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		_ = coderdtest.CreateFirstUser(t, client)

		client.SessionToken = ""
		_, err := client.CreateAPIKey(context.Background(), codersdk.Me)
		var apiErr *codersdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusUnauthorized, apiErr.StatusCode())
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		_ = coderdtest.CreateFirstUser(t, client)
		apiKey, err := client.CreateAPIKey(context.Background(), codersdk.Me)
		require.NotNil(t, apiKey)
		require.GreaterOrEqual(t, len(apiKey.Key), 2)
		require.NoError(t, err)
	})
}

func TestPostWorkspacesByUser(t *testing.T) {
	t.Parallel()
	t.Run("InvalidTemplate", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		_ = coderdtest.CreateFirstUser(t, client)
		_, err := client.CreateWorkspace(context.Background(), codersdk.Me, codersdk.CreateWorkspaceRequest{
			TemplateID: uuid.New(),
			Name:       "workspace",
		})
		require.Error(t, err)
		var apiErr *codersdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusBadRequest, apiErr.StatusCode())
	})

	t.Run("NoTemplateAccess", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		first := coderdtest.CreateFirstUser(t, client)

		other := coderdtest.CreateAnotherUser(t, client, first.OrganizationID)
		org, err := other.CreateOrganization(context.Background(), codersdk.Me, codersdk.CreateOrganizationRequest{
			Name: "another",
		})
		require.NoError(t, err)
		version := coderdtest.CreateTemplateVersion(t, other, org.ID, nil)
		template := coderdtest.CreateTemplate(t, other, org.ID, version.ID)

		_, err = client.CreateWorkspace(context.Background(), codersdk.Me, codersdk.CreateWorkspaceRequest{
			TemplateID: template.ID,
			Name:       "workspace",
		})
		require.Error(t, err)
		var apiErr *codersdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusUnauthorized, apiErr.StatusCode())
	})

	t.Run("AlreadyExists", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		coderdtest.NewProvisionerDaemon(t, client)
		user := coderdtest.CreateFirstUser(t, client)
		version := coderdtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := coderdtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		coderdtest.AwaitTemplateVersionJob(t, client, version.ID)
		workspace := coderdtest.CreateWorkspace(t, client, codersdk.Me, template.ID)
		_, err := client.CreateWorkspace(context.Background(), codersdk.Me, codersdk.CreateWorkspaceRequest{
			TemplateID: template.ID,
			Name:       workspace.Name,
		})
		require.Error(t, err)
		var apiErr *codersdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusConflict, apiErr.StatusCode())
	})

	t.Run("Create", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		coderdtest.NewProvisionerDaemon(t, client)
		user := coderdtest.CreateFirstUser(t, client)
		version := coderdtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		template := coderdtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		coderdtest.AwaitTemplateVersionJob(t, client, version.ID)
		_ = coderdtest.CreateWorkspace(t, client, codersdk.Me, template.ID)
	})
}

func TestWorkspacesByUser(t *testing.T) {
	t.Parallel()
	t.Run("ListEmpty", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		coderdtest.CreateFirstUser(t, client)
		_, err := client.WorkspacesByUser(context.Background(), codersdk.Me)
		require.NoError(t, err)
	})
	t.Run("List", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		coderdtest.NewProvisionerDaemon(t, client)
		user := coderdtest.CreateFirstUser(t, client)
		version := coderdtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		coderdtest.AwaitTemplateVersionJob(t, client, version.ID)
		template := coderdtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		_ = coderdtest.CreateWorkspace(t, client, codersdk.Me, template.ID)
		workspaces, err := client.WorkspacesByUser(context.Background(), codersdk.Me)
		require.NoError(t, err)
		require.Len(t, workspaces, 1)
	})
}

func TestWorkspaceByUserAndName(t *testing.T) {
	t.Parallel()
	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		coderdtest.CreateFirstUser(t, client)
		_, err := client.WorkspaceByName(context.Background(), codersdk.Me, "something")
		var apiErr *codersdk.Error
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, http.StatusNotFound, apiErr.StatusCode())
	})
	t.Run("Get", func(t *testing.T) {
		t.Parallel()
		client := coderdtest.New(t, nil)
		coderdtest.NewProvisionerDaemon(t, client)
		user := coderdtest.CreateFirstUser(t, client)
		version := coderdtest.CreateTemplateVersion(t, client, user.OrganizationID, nil)
		coderdtest.AwaitTemplateVersionJob(t, client, version.ID)
		template := coderdtest.CreateTemplate(t, client, user.OrganizationID, version.ID)
		workspace := coderdtest.CreateWorkspace(t, client, codersdk.Me, template.ID)
		_, err := client.WorkspaceByName(context.Background(), codersdk.Me, workspace.Name)
		require.NoError(t, err)
	})
}
