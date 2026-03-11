package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tk-425/Codefind/internal/authflow"
	"github.com/tk-425/Codefind/internal/config"
	"github.com/tk-425/Codefind/internal/keychain"
	"github.com/tk-425/Codefind/pkg/api"
)

func newAuthCommand(configPath *string) *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage CLI authentication",
	}

	authCmd.AddCommand(newAuthLoginCommand(configPath))
	authCmd.AddCommand(newAuthLogoutCommand(configPath))
	authCmd.AddCommand(newAuthStatusCommand(configPath))

	return authCmd
}

func newOrgCommand(configPath *string) *cobra.Command {
	orgCmd := &cobra.Command{
		Use:   "org",
		Short: "Inspect organization access for the current token",
	}
	orgCmd.AddCommand(newOrgListCommand(configPath))
	return orgCmd
}

func newOrgListCommand(configPath *string) *cobra.Command {
	var jsonOutput bool

	command := &cobra.Command{
		Use:   "list",
		Short: "List organizations available to the authenticated user",
		RunE: func(cmd *cobra.Command, _ []string) error {
			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath, jsonOutput)
			if err != nil {
				return err
			}

			response, err := apiClient.GetOrganizations(cmd.Context())
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd.OutOrStdout(), jsonOutput, response, func(stdout io.Writer) error {
				return writeOrgListOutput(stdout, response)
			})
		},
	}

	addJSONFlag(command, &jsonOutput)
	return command
}

func newAdminCommand(configPath *string) *cobra.Command {
	adminCmd := &cobra.Command{
		Use:   "admin",
		Short: "Manage organization members and invitations for the current token org",
	}
	adminCmd.AddCommand(newAdminListCommand(configPath))
	adminCmd.AddCommand(newAdminInviteCommand(configPath))
	adminCmd.AddCommand(newAdminRevokeInviteCommand(configPath))
	adminCmd.AddCommand(newAdminRemoveCommand(configPath))
	return adminCmd
}

func newAdminListCommand(configPath *string) *cobra.Command {
	var jsonOutput bool

	command := &cobra.Command{
		Use:   "list",
		Short: "List members and invitations for the active token organization",
		RunE: func(cmd *cobra.Command, _ []string) error {
			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath, jsonOutput)
			if err != nil {
				return err
			}

			members, err := apiClient.GetAdminMembers(cmd.Context())
			if err != nil {
				return err
			}
			invitations, err := apiClient.GetAdminInvitations(cmd.Context())
			if err != nil {
				return err
			}
			payload := map[string]any{
				"members":     members,
				"invitations": invitations,
			}
			return writeCommandOutput(cmd.OutOrStdout(), jsonOutput, payload, func(stdout io.Writer) error {
				return writeAdminListOutput(stdout, members, invitations)
			})
		},
	}

	addJSONFlag(command, &jsonOutput)
	return command
}

func newAdminInviteCommand(configPath *string) *cobra.Command {
	var (
		email string
		role  string
	)

	command := &cobra.Command{
		Use:   "invite",
		Short: "Invite a user into the current token organization",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(email) == "" {
				return errors.New("--email is required")
			}
			if role != "org:admin" && role != "org:member" {
				return errors.New("--role must be org:admin or org:member")
			}

			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath, false)
			if err != nil {
				return err
			}

			response, err := apiClient.CreateAdminInvitation(cmd.Context(), api.CreateOrganizationInvitationRequest{
				EmailAddress: email,
				Role:         role,
			})
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), response)
		},
	}
	command.Flags().StringVar(&email, "email", "", "email address to invite")
	command.Flags().StringVar(&role, "role", "org:member", "organization role for the invitee")
	return command
}

func newAdminRevokeInviteCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "revoke-invite <invitation-id>",
		Short: "Revoke a pending invitation in the current token organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath, false)
			if err != nil {
				return err
			}

			response, err := apiClient.RevokeAdminInvitation(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), response)
		},
	}
}

func newAdminRemoveCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <user-id>",
		Short: "Remove a member from the current token organization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := loadAuthenticatedClient(cmd.Context(), cmd.OutOrStdout(), *configPath, false)
			if err != nil {
				return err
			}

			response, err := apiClient.RemoveAdminMember(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), response)
		},
	}
}

func newAuthLoginCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Open the browser and authenticate with Clerk",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return browserLoginRunner(cmd.Context(), cmd.OutOrStdout(), *configPath)
		},
	}
}

func newAuthLogoutCommand(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Delete the stored CLI token",
		RunE: func(cmd *cobra.Command, _ []string) error {
			manager := defaultTokenManager()
			err := manager.DeleteToken()
			if err != nil && !errors.Is(err, keychain.ErrNotFound) {
				return err
			}

			cfg, loadErr := config.LoadOrDefault(*configPath)
			if loadErr != nil {
				return loadErr
			}
			cfg.ActiveOrgID = ""
			if saveErr := config.Save(*configPath, cfg); saveErr != nil {
				return saveErr
			}

			if errors.Is(err, keychain.ErrNotFound) {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "no stored token was present")
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "stored token deleted")
			return err
		},
	}
}

func newAuthStatusCommand(configPath *string) *cobra.Command {
	var jsonOutput bool

	command := &cobra.Command{
		Use:   "status",
		Short: "Show CLI authentication status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.LoadOrDefault(*configPath)
			if err != nil {
				return err
			}

			status := map[string]any{
				"authenticated": false,
				"active_org_id": cfg.DisplayMap()["active_org_id"],
			}

			token, err := defaultTokenManager().LoadToken()
			if err != nil {
				if errors.Is(err, keychain.ErrNotFound) {
					return writeCommandOutput(cmd.OutOrStdout(), jsonOutput, status, func(stdout io.Writer) error {
						return writeAuthStatusOutput(stdout, status)
					})
				}
				return err
			}

			status["authenticated"] = true
			if claims, err := authflow.DecodeTokenClaims(token); err == nil {
				if claims.OrgID != "" {
					status["token_org_id"] = claims.OrgID
				}
			}
			if expiry, err := authflow.TokenExpiryTime(token); err == nil {
				status["expires_at"] = expiry.Format(time.RFC3339)
				status["expired"] = time.Now().UTC().After(expiry)
			}

			return writeCommandOutput(cmd.OutOrStdout(), jsonOutput, status, func(stdout io.Writer) error {
				return writeAuthStatusOutput(stdout, status)
			})
		},
	}

	addJSONFlag(command, &jsonOutput)
	return command
}

func runBrowserLogin(ctx context.Context, stdout io.Writer, configPath string) error {
	cfg, err := loadRequiredConfig(configPath)
	if err != nil {
		return err
	}
	if cfg.ServerURL == "" {
		return errors.New("server_url is not configured; run 'codefind config --server-url <url>'")
	}
	webAppURL := cfg.WebAppURL
	if webAppURL == "" {
		webAppURL = "http://localhost:5173"
	}

	listener, err := newCallbackListener()
	if err != nil {
		return err
	}
	defer listener.Close()

	timeoutCtx, cancel := context.WithTimeout(ctx, authflow.LoginTimeout())
	defer cancel()

	redirectURI, waitForToken, err := startCallbackServer(timeoutCtx, listener, webAppURL)
	if err != nil {
		return err
	}

	signInURL, err := buildSignInURL(cfg.ServerURL, redirectURI)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(stdout, "opening browser: %s\n", signInURL); err != nil {
		return err
	}
	if err := openBrowser(signInURL); err != nil {
		return fmt.Errorf("open browser manually with %s: %w", signInURL, err)
	}

	token, err := waitForToken()
	if err != nil {
		return err
	}

	manager := defaultTokenManager()
	if err := manager.SaveToken(token); err != nil {
		return err
	}

	if claims, err := authflow.DecodeTokenClaims(token); err == nil {
		cfg.ActiveOrgID = claims.OrgID
	} else {
		cfg.ActiveOrgID = ""
	}
	if err := config.Save(configPath, cfg); err != nil {
		return err
	}

	_, err = fmt.Fprintln(stdout, "authentication stored in keychain")
	return err
}
