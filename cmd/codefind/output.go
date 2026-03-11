package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/tk-425/Codefind/pkg/api"
	"golang.org/x/term"
)

const querySnippetLimit = 160

type outputStyles struct {
	header    func(a ...any) string
	label     func(a ...any) string
	ok        func(a ...any) string
	warn      func(a ...any) string
	error     func(a ...any) string
	muted     func(a ...any) string
	accent    func(a ...any) string
	path      func(a ...any) string
	value     func(a ...any) string
	highlight func(a ...any) string
}

func newOutputStyles(stdout io.Writer) outputStyles {
	colorEnabled := shouldColorize(stdout)
	newSprint := func(attrs ...color.Attribute) func(a ...any) string {
		return color.New(attrs...).SprintFunc()
	}

	styles := outputStyles{
		header:    newSprint(color.FgGreen, color.Bold),
		label:     newSprint(color.FgCyan, color.Bold),
		ok:        newSprint(color.FgGreen),
		warn:      newSprint(color.FgYellow),
		error:     newSprint(color.FgRed),
		muted:     newSprint(color.FgHiBlack),
		accent:    newSprint(color.FgMagenta, color.Bold),
		path:      newSprint(color.FgWhite, color.Bold),
		value:     fmt.Sprint,
		highlight: newSprint(color.FgRed),
	}

	if colorEnabled {
		return styles
	}

	plain := func(a ...any) string { return fmt.Sprint(a...) }
	return outputStyles{
		header:    plain,
		label:     plain,
		ok:        plain,
		warn:      plain,
		error:     plain,
		muted:     plain,
		accent:    plain,
		path:      plain,
		value:     plain,
		highlight: plain,
	}
}

func shouldColorize(stdout io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("CLICOLOR_FORCE") != "" || os.Getenv("FORCE_COLOR") != "" {
		return true
	}
	file, ok := stdout.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func addJSONFlag(command *cobra.Command, target *bool) {
	command.Flags().BoolVar(target, "json", false, "print structured JSON output")
}

func writeCommandOutput(stdout io.Writer, jsonOutput bool, value any, render func(io.Writer) error) error {
	if jsonOutput {
		return writeJSON(stdout, value)
	}
	return render(stdout)
}

func writeCollectionListOutput(stdout io.Writer, response api.CollectionListResponse) error {
	styles := newOutputStyles(stdout)
	if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.header("Indexed repos:"), styles.value(response.TotalCount)); err != nil {
		return err
	}
	if len(response.Data) == 0 {
		_, err := fmt.Fprintln(stdout, styles.muted("No indexed repos found."))
		return err
	}
	for _, repo := range response.Data {
		if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.accent("•"), styles.path(repo.RepoID)); err != nil {
			return err
		}
	}
	return nil
}

func writeOrgListOutput(stdout io.Writer, response api.OrgListResponse) error {
	styles := newOutputStyles(stdout)
	if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.header("Organizations:"), styles.value(response.TotalCount)); err != nil {
		return err
	}
	if len(response.Data) == 0 {
		_, err := fmt.Fprintln(stdout, styles.muted("No organizations found."))
		return err
	}
	for _, org := range response.Data {
		name := org.OrganizationName
		if strings.TrimSpace(name) == "" {
			name = org.OrganizationID
		}
		if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.accent("•"), styles.path(name)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(stdout, "  %s %s\n", styles.muted("id:"), styles.muted(org.OrganizationID)); err != nil {
			return err
		}
		if strings.TrimSpace(org.OrganizationSlug) != "" {
			if _, err := fmt.Fprintf(stdout, "  %s %s\n", styles.muted("slug:"), styles.muted(org.OrganizationSlug)); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(stdout, "  %s %s\n", styles.label("role:"), styles.value(org.Role)); err != nil {
			return err
		}
	}
	return nil
}

func writeAdminListOutput(stdout io.Writer, members api.OrganizationMemberListResponse, invitations api.OrganizationInvitationListResponse) error {
	styles := newOutputStyles(stdout)
	if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.header("Members:"), styles.value(members.TotalCount)); err != nil {
		return err
	}
	if len(members.Data) == 0 {
		if _, err := fmt.Fprintln(stdout, styles.muted("No members found.")); err != nil {
			return err
		}
	} else {
		for _, member := range members.Data {
			name := strings.TrimSpace(strings.Join([]string{member.FirstName, member.LastName}, " "))
			if name == "" {
				name = member.EmailAddress
			}
			if name == "" {
				name = member.UserID
			}
			if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.accent("•"), styles.path(name)); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(stdout, "  %s %s\n", styles.muted("user:"), styles.muted(member.UserID)); err != nil {
				return err
			}
			if member.EmailAddress != "" {
				if _, err := fmt.Fprintf(stdout, "  %s %s\n", styles.muted("email:"), styles.muted(member.EmailAddress)); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(stdout, "  %s %s\n", styles.label("role:"), styles.value(member.Role)); err != nil {
				return err
			}
		}
	}

	if _, err := fmt.Fprintf(stdout, "\n%s %s\n", styles.header("Invitations:"), styles.value(invitations.TotalCount)); err != nil {
		return err
	}
	if len(invitations.Data) == 0 {
		_, err := fmt.Fprintln(stdout, styles.muted("No invitations found."))
		return err
	}
	for _, invitation := range invitations.Data {
		if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.accent("•"), styles.path(invitation.EmailAddress)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(stdout, "  %s %s\n", styles.muted("id:"), styles.muted(invitation.InvitationID)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(stdout, "  %s %s\n", styles.label("role:"), styles.value(invitation.Role)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(stdout, "  %s %s\n", styles.label("status:"), formatStatusValue(styles, invitation.Status)); err != nil {
			return err
		}
	}
	return nil
}

func writeAuthStatusOutput(stdout io.Writer, status map[string]any) error {
	styles := newOutputStyles(stdout)
	authenticated, _ := status["authenticated"].(bool)
	authLabel := styles.error("NO")
	if authenticated {
		authLabel = styles.ok("YES")
	}
	if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.header("Authenticated:"), authLabel); err != nil {
		return err
	}
	if activeOrgID, ok := status["active_org_id"]; ok {
		if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.label("Active Org:"), styles.value(fmt.Sprint(activeOrgID))); err != nil {
			return err
		}
	}
	if tokenOrgID, ok := status["token_org_id"]; ok {
		if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.label("Token Org:"), styles.value(fmt.Sprint(tokenOrgID))); err != nil {
			return err
		}
	}
	if expiresAt, ok := status["expires_at"]; ok {
		if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.label("Expires At:"), styles.value(fmt.Sprint(expiresAt))); err != nil {
			return err
		}
	}
	if expired, ok := status["expired"]; ok {
		if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.label("Expired:"), formatBoolValue(styles, expired)); err != nil {
			return err
		}
	}
	return nil
}

func writeHealthOutput(stdout io.Writer, response api.HealthResponse) error {
	styles := newOutputStyles(stdout)
	if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.label("Server:"), formatStatusValue(styles, response.Status)); err != nil {
		return err
	}
	if response.OllamaStatus != "" {
		if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.label("Ollama:"), formatStatusValue(styles, response.OllamaStatus)); err != nil {
			return err
		}
	}
	if response.QdrantStatus != "" {
		if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.label("Qdrant:"), formatStatusValue(styles, response.QdrantStatus)); err != nil {
			return err
		}
	}
	if strings.TrimSpace(response.Timestamp) != "" {
		if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.muted("Timestamp:"), styles.muted(response.Timestamp)); err != nil {
			return err
		}
	}
	return nil
}

func writeLSPStatusOutput(stdout io.Writer, response lspStatusResponse) error {
	styles := newOutputStyles(stdout)
	if _, err := fmt.Fprintf(stdout, "%s %s/%s\n", styles.header("LSP availability:"), styles.ok(response.AvailableCount), styles.value(response.SupportedCount)); err != nil {
		return err
	}
	if len(response.Languages) == 0 {
		_, err := fmt.Fprintln(stdout, styles.muted("No supported language servers configured."))
		return err
	}
	for _, language := range response.Languages {
		statusLabel := "missing"
		statusStyle := styles.error
		if language.Available {
			statusLabel = "available"
			statusStyle = styles.ok
		}
		if _, err := fmt.Fprintf(stdout, "%s %s: %s %s\n", styles.accent("•"), styles.path(language.Language), statusStyle(statusLabel), styles.muted("("+language.Executable+")")); err != nil {
			return err
		}
		if strings.TrimSpace(language.Path) != "" {
			if _, err := fmt.Fprintf(stdout, "  %s %s\n", styles.muted("path:"), styles.muted(language.Path)); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeStatsOutput(stdout io.Writer, response api.StatsResponse) error {
	styles := newOutputStyles(stdout)
	scopeLabel := "current project"
	if strings.TrimSpace(response.RepoID) != "" {
		scopeLabel = "repo " + response.RepoID
	} else if len(response.Repos) == 1 && strings.TrimSpace(response.Repos[0].RepoID) != "" {
		scopeLabel = "repo " + response.Repos[0].RepoID
	}
	if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.header("Stats for"), styles.path(scopeLabel)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.label("Active Chunks:"), styles.value(response.ActiveChunks)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.label("Deleted Chunks:"), styles.value(response.DeletedChunks)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.label("Total Chunks:"), styles.value(response.TotalChunks)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "%s %s\n", styles.label("Storage Overhead:"), styles.highlight(fmt.Sprintf("%.1f%%", response.OverheadPercent))); err != nil {
		return err
	}
	if response.OverheadPercent > 15 {
		if _, err := fmt.Fprintf(stdout, "\n%s %s\n", styles.warn("Consider running:"), styles.value("codefind cleanup --older-than=30")); err != nil {
			return err
		}
	}
	return nil
}

func writeQueryOutput(stdout io.Writer, response api.QueryResponse) error {
	styles := newOutputStyles(stdout)
	if _, err := fmt.Fprintf(stdout, "%s %s", styles.header("Results:"), styles.value(response.TotalCount)); err != nil {
		return err
	}
	if response.Page > 0 && response.PageSize > 0 {
		if _, err := fmt.Fprintf(stdout, " %s", styles.muted(fmt.Sprintf("(page %d, page size %d)", response.Page, response.PageSize))); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(stdout); err != nil {
		return err
	}
	if len(response.Data) == 0 {
		_, err := fmt.Fprintln(stdout, styles.muted("No matches found."))
		return err
	}
	for index, result := range response.Data {
		if _, err := fmt.Fprintf(stdout, "\n%s %s\n", styles.accent(fmt.Sprintf("%d.", index+1)), styles.path(formatQueryHeading(result))); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(stdout, "   %s %s\n", styles.label("Score:"), styles.highlight(fmt.Sprintf("%.3f", result.Score))); err != nil {
			return err
		}
		if repoLine := formatQueryMetaLine(result); repoLine != "" {
			if _, err := fmt.Fprintf(stdout, "   %s\n", styles.muted(repoLine)); err != nil {
				return err
			}
		}
		if snippet := formatQuerySnippet(result); snippet != "" {
			if _, err := fmt.Fprintf(stdout, "   %s\n", styles.value(snippet)); err != nil {
				return err
			}
		}
	}
	if response.HasMore {
		_, err := fmt.Fprintf(stdout, "\n%s %s\n", styles.warn("More results available."), styles.muted("Use --page to continue."))
		return err
	}
	return nil
}

func formatStatusValue(styles outputStyles, value string) string {
	upper := strings.ToUpper(value)
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ok", "available", "healthy":
		return styles.ok(upper)
	case "degraded", "unknown":
		return styles.warn(upper)
	case "missing", "unavailable", "error", "failed":
		return styles.error(upper)
	default:
		return styles.value(upper)
	}
}

func formatBoolValue(styles outputStyles, value any) string {
	boolean, ok := value.(bool)
	if !ok {
		return styles.value(fmt.Sprint(value))
	}
	if boolean {
		return styles.warn("YES")
	}
	return styles.ok("NO")
}

func formatQueryHeading(result api.QueryResult) string {
	path := strings.TrimSpace(result.Path)
	if path == "" {
		path = "(no path)"
	}
	if lineRange := formatLineRange(result.StartLine, result.EndLine); lineRange != "" {
		return fmt.Sprintf("%s:%s", path, lineRange)
	}
	return path
}

func formatQueryMetaLine(result api.QueryResult) string {
	parts := make([]string, 0, 3)
	if repoID := strings.TrimSpace(result.RepoID); repoID != "" {
		parts = append(parts, "repo "+repoID)
	}
	if project := strings.TrimSpace(result.Project); project != "" {
		parts = append(parts, "project "+project)
	}
	if language := strings.TrimSpace(result.Language); language != "" {
		parts = append(parts, "lang "+language)
	}
	return strings.Join(parts, " | ")
}

func formatQuerySnippet(result api.QueryResult) string {
	snippet := strings.TrimSpace(result.Snippet)
	if snippet == "" {
		snippet = strings.TrimSpace(result.Content)
	}
	if snippet == "" {
		return ""
	}
	snippet = strings.Join(strings.Fields(snippet), " ")
	if len(snippet) > querySnippetLimit {
		snippet = snippet[:querySnippetLimit-3] + "..."
	}
	return snippet
}

func formatLineRange(startLine, endLine int) string {
	switch {
	case startLine > 0 && endLine > 0 && startLine != endLine:
		return fmt.Sprintf("%d-%d", startLine, endLine)
	case startLine > 0:
		return fmt.Sprintf("%d", startLine)
	case endLine > 0:
		return fmt.Sprintf("%d", endLine)
	default:
		return ""
	}
}
