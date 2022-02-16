package cli

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os/exec"
	"os/user"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/go-playground/validator/v10"
	"github.com/manifoldco/promptui"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"github.com/coder/coder/coderd"
	"github.com/coder/coder/codersdk"
)

const (
	goosWindows = "windows"
	goosLinux   = "linux"
	goosDarwin  = "darwin"
)

func login() *cobra.Command {
	return &cobra.Command{
		Use:  "login <url>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rawURL := args[0]

			if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
				scheme := "https"
				if strings.HasPrefix(rawURL, "localhost") {
					scheme = "http"
				}
				rawURL = fmt.Sprintf("%s://%s", scheme, rawURL)
			}
			serverURL, err := url.Parse(rawURL)
			if err != nil {
				return xerrors.Errorf("parse raw url %q: %w", rawURL, err)
			}
			// Default to HTTPs. Enables simple URLs like: master.cdr.dev
			if serverURL.Scheme == "" {
				serverURL.Scheme = "https"
			}

			client := codersdk.New(serverURL)
			hasInitialUser, err := client.HasInitialUser(cmd.Context())
			if err != nil {
				return xerrors.Errorf("has initial user: %w", err)
			}
			if !hasInitialUser {
				if !isTTY(cmd) {
					return xerrors.New("the initial user cannot be created in non-interactive mode. use the API")
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s Your Coder deployment hasn't been set up!\n", color.HiBlackString(">"))

				_, err := prompt(cmd, &promptui.Prompt{
					Label:     "Would you like to create the first user?",
					IsConfirm: true,
					Default:   "y",
				})
				if err != nil {
					return xerrors.Errorf("create user prompt: %w", err)
				}
				currentUser, err := user.Current()
				if err != nil {
					return xerrors.Errorf("get current user: %w", err)
				}
				username, err := prompt(cmd, &promptui.Prompt{
					Label:   "What username would you like?",
					Default: currentUser.Username,
				})
				if err != nil {
					return xerrors.Errorf("pick username prompt: %w", err)
				}

				organization, err := prompt(cmd, &promptui.Prompt{
					Label:   "What is the name of your organization?",
					Default: "acme-corp",
				})
				if err != nil {
					return xerrors.Errorf("pick organization prompt: %w", err)
				}

				email, err := prompt(cmd, &promptui.Prompt{
					Label: "What's your email?",
					Validate: func(s string) error {
						err := validator.New().Var(s, "email")
						if err != nil {
							return xerrors.New("That's not a valid email address!")
						}
						return err
					},
				})
				if err != nil {
					return xerrors.Errorf("specify email prompt: %w", err)
				}

				password, err := prompt(cmd, &promptui.Prompt{
					Label: "Enter a password:",
					Mask:  '*',
				})
				if err != nil {
					return xerrors.Errorf("specify password prompt: %w", err)
				}

				_, err = client.CreateInitialUser(cmd.Context(), coderd.CreateInitialUserRequest{
					Email:        email,
					Username:     username,
					Password:     password,
					Organization: organization,
				})
				if err != nil {
					return xerrors.Errorf("create initial user: %w", err)
				}
				resp, err := client.LoginWithPassword(cmd.Context(), coderd.LoginWithPasswordRequest{
					Email:    email,
					Password: password,
				})
				if err != nil {
					return xerrors.Errorf("login with password: %w", err)
				}

				err = saveSessionToken(cmd, client, resp.SessionToken, serverURL)
				if err != nil {
					return xerrors.Errorf("save session token: %w", err)
				}

				return nil
			}

			authURL := *serverURL
			authURL.Path = serverURL.Path + "/cli-auth"
			if err := openURL(authURL.String()); err != nil {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Open the following in your browser:\n\n\t%s\n\n", authURL.String())
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Your browser has been opened to visit:\n\n\t%s\n\n", authURL.String())
			}

			apiKey, err := prompt(cmd, &promptui.Prompt{
				Label: "Paste your token here:",
				Validate: func(token string) error {
					client.SessionToken = token
					_, err := client.User(cmd.Context(), "me")
					if err != nil {
						return xerrors.New("That's not a valid token!")
					}
					return err
				},
			})
			if err != nil {
				return xerrors.Errorf("specify email prompt: %w", err)
			}

			err = saveSessionToken(cmd, client, apiKey, serverURL)
			if err != nil {
				return xerrors.Errorf("save session token after login: %w", err)
			}

			return nil
		},
	}
}

func saveSessionToken(cmd *cobra.Command, client *codersdk.Client, sessionToken string, serverURL *url.URL) error {
	// Login to get user data - verify it is OK before persisting
	client.SessionToken = sessionToken
	resp, err := client.User(cmd.Context(), "me")
	if err != nil {
		return xerrors.Errorf("get user: ", err)
	}

	config := createConfig(cmd)
	err = config.Session().Write(sessionToken)
	if err != nil {
		return xerrors.Errorf("write session token: %w", err)
	}
	err = config.URL().Write(serverURL.String())
	if err != nil {
		return xerrors.Errorf("write server url: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s Welcome to Coder, %s! You're authenticated.\n", color.HiBlackString(">"), color.HiCyanString(resp.Username))
	return nil
}

// isWSL determines if coder-cli is running within Windows Subsystem for Linux
func isWSL() (bool, error) {
	if runtime.GOOS == goosDarwin || runtime.GOOS == goosWindows {
		return false, nil
	}
	data, err := ioutil.ReadFile("/proc/version")
	if err != nil {
		return false, xerrors.Errorf("read /proc/version: %w", err)
	}
	return strings.Contains(strings.ToLower(string(data)), "microsoft"), nil
}

// openURL opens the provided URL via user's default browser
func openURL(url string) error {
	var cmd string
	var args []string

	wsl, err := isWSL()
	if err != nil {
		return xerrors.Errorf("test running Windows Subsystem for Linux: %w", err)
	}

	if wsl {
		cmd = "cmd.exe"
		args = []string{"/c", "start"}
		url = strings.ReplaceAll(url, "&", "^&")
		args = append(args, url)
		return exec.Command(cmd, args...).Start()
	}

	return browser.OpenURL(url)
}
