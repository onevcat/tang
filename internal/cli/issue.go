package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"tangled.org/onev.cat/tang/internal/auth"
	"tangled.org/onev.cat/tang/internal/config"
	"tangled.org/onev.cat/tang/internal/repo"
	"tangled.org/onev.cat/tang/internal/tangled"
)

type issueViewOutput struct {
	tangled.Issue
	Comments []tangled.Comment `json:"comments"`
}

const issueAtprotoHelp = "By default, numeric issue arguments are Tangled AppView issue numbers. With --atproto, numeric IDs are rejected; pass an issue rkey, unique rkey prefix, or full AT URI."

func newIssueCommand(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Manage Tangled issues",
	}
	cmd.AddCommand(newIssueListCommand(opts))
	cmd.AddCommand(newIssueCreateCommand(opts))
	cmd.AddCommand(newIssueViewCommand(opts))
	cmd.AddCommand(newIssueStateCommand(opts, "close", "closed"))
	cmd.AddCommand(newIssueStateCommand(opts, "reopen", "open"))
	cmd.AddCommand(newIssueEditCommand(opts))
	cmd.AddCommand(newIssueCommentCommand(opts))
	return cmd
}

func newIssueListCommand(opts *RootOptions) *cobra.Command {
	var state string
	var limit int
	var atproto bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List issues",
		Long:  "List issues from PDS/Constellation. Default output uses stable rkeys; --atproto makes the raw-record mode explicit for scripting and debugging AppView projection gaps.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, service, repoURI, err := issueDependencies(cmd)
			_ = cfg
			if err != nil {
				return err
			}
			issues, err := service.ListIssues(cmd.Context(), repoURI, tangled.IssueListOptions{State: state, Limit: limit})
			if err != nil {
				return err
			}
			if rendered, err := renderJSONIfRequested(cmd, opts, issues); rendered || err != nil {
				return err
			}
			for _, issue := range issues {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", issueDisplayID(issue), issue.Title, issue.State, issue.Author); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&state, "state", "open", "Filter by state: open, closed, all")
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum issues to list")
	cmd.Flags().BoolVar(&atproto, "atproto", false, "Show raw ATProto issue records from PDS/Constellation")
	return cmd
}

func newIssueCreateCommand(opts *RootOptions) *cobra.Command {
	flags := bodyFlags{}
	cmd := &cobra.Command{
		Use:   "create <title>",
		Short: "Create an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := auth.Load()
			if err != nil {
				return err
			}
			_, service, repoURI, err := issueDependencies(cmd)
			if err != nil {
				return err
			}
			body, _, err := readBodyInput(flags, cmd.InOrStdin())
			if err != nil {
				return err
			}
			issue, err := service.CreateIssue(cmd.Context(), session, repoURI, args[0], body)
			if err != nil {
				return err
			}
			if rendered, err := renderJSONIfRequested(cmd, opts, issue); rendered || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created issue %s\n", tangled.RKeyFromURI(issue.URI))
			return err
		},
	}
	addBodyFlags(cmd, &flags)
	return cmd
}

func newIssueViewCommand(opts *RootOptions) *cobra.Command {
	var web bool
	var atproto bool
	cmd := &cobra.Command{
		Use:   "view <issue>",
		Short: "View an issue",
		Long:  "View an issue.\n\n" + issueAtprotoHelp,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, service, repoURI, err := issueDependencies(cmd)
			if err != nil {
				return err
			}
			issue, err := resolveIssueArg(cmd, service, repoURI, args[0], atproto)
			if err != nil {
				return err
			}
			if web {
				context, err := currentRepoContext(cmd, cfg)
				if err != nil {
					return err
				}
				target := issueURL(cfg, context, issue)
				if target == "" {
					return fmt.Errorf("issue %s has no AppView number; open it by AT URI outside tang", issueDisplayID(issue))
				}
				return openBrowserForCLI(target)
			}
			comments, _ := service.ListComments(cmd.Context(), issue.URI)
			if rendered, err := renderJSONIfRequested(cmd, opts, issueViewOutput{Issue: issue, Comments: comments}); rendered || err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Issue %s %s\nTitle: %s\nAuthor: %s\nCreated: %s\nURI: %s\n\n", issueDisplayID(issue), issue.State, issue.Title, issue.Author, issue.CreatedAt, issue.URI)
			if issue.Body != "" {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n", issue.Body)
			}
			for _, comment := range comments {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Comment by %s at %s\n%s\n\n", comment.Author, comment.CreatedAt, comment.Body); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&web, "web", false, "Open the issue in a browser")
	cmd.Flags().BoolVar(&atproto, "atproto", false, "Use raw PDS/Constellation resolution; numeric AppView IDs are not accepted")
	return cmd
}

func newIssueStateCommand(opts *RootOptions, name, state string) *cobra.Command {
	var atproto bool
	cmd := &cobra.Command{
		Use:   name + " <issue>",
		Short: name + " an issue",
		Long:  strings.ToUpper(name[:1]) + name[1:] + " an issue.\n\n" + issueAtprotoHelp,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := auth.Load()
			if err != nil {
				return err
			}
			_, service, repoURI, err := issueDependencies(cmd)
			if err != nil {
				return err
			}
			issue, err := resolveIssueArg(cmd, service, repoURI, args[0], atproto)
			if err != nil {
				return err
			}
			if err := service.SetIssueState(cmd.Context(), session, issue.URI, state); err != nil {
				return err
			}
			issue.State = state
			if rendered, err := renderJSONIfRequested(cmd, opts, issue); rendered || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Issue %s is now %s\n", issueDisplayID(issue), state)
			return err
		},
	}
	cmd.Flags().BoolVar(&atproto, "atproto", false, "Use raw PDS/Constellation resolution; numeric AppView IDs are not accepted")
	return cmd
}

func newIssueEditCommand(opts *RootOptions) *cobra.Command {
	var title string
	var atproto bool
	flags := bodyFlags{}
	cmd := &cobra.Command{
		Use:   "edit <issue>",
		Short: "Edit an issue",
		Long:  "Edit an issue.\n\n" + issueAtprotoHelp,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := auth.Load()
			if err != nil {
				return err
			}
			_, service, repoURI, err := issueDependencies(cmd)
			if err != nil {
				return err
			}
			body, hasBody, err := readBodyInput(flags, cmd.InOrStdin())
			if err != nil {
				return err
			}
			if title == "" && !hasBody {
				return fmt.Errorf("at least one of --title, --body, or --body-file is required")
			}
			issue, err := resolveIssueArg(cmd, service, repoURI, args[0], atproto)
			if err != nil {
				return err
			}
			updated, err := service.UpdateIssue(cmd.Context(), session, issue.URI, title, body, title != "", hasBody)
			if err != nil {
				return err
			}
			if rendered, err := renderJSONIfRequested(cmd, opts, updated); rendered || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Updated issue %s\n", tangled.RKeyFromURI(updated.URI))
			return err
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "New title")
	cmd.Flags().BoolVar(&atproto, "atproto", false, "Use raw PDS/Constellation resolution; numeric AppView IDs are not accepted")
	addBodyFlags(cmd, &flags)
	return cmd
}

func newIssueCommentCommand(opts *RootOptions) *cobra.Command {
	var atproto bool
	flags := bodyFlags{}
	cmd := &cobra.Command{
		Use:   "comment <issue>",
		Short: "Add a comment to an issue",
		Long:  "Add a comment to an issue.\n\n" + issueAtprotoHelp,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := auth.Load()
			if err != nil {
				return err
			}
			_, service, repoURI, err := issueDependencies(cmd)
			if err != nil {
				return err
			}
			body, hasBody, err := readBodyInput(flags, cmd.InOrStdin())
			if err != nil {
				return err
			}
			if !hasBody || body == "" {
				return fmt.Errorf("comment body is required")
			}
			issue, err := resolveIssueArg(cmd, service, repoURI, args[0], atproto)
			if err != nil {
				return err
			}
			comment, err := service.AddComment(cmd.Context(), session, issue.URI, body)
			if err != nil {
				return err
			}
			if rendered, err := renderJSONIfRequested(cmd, opts, comment); rendered || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Commented on issue %s\n", issueDisplayID(issue))
			return err
		},
	}
	cmd.Flags().BoolVar(&atproto, "atproto", false, "Use raw PDS/Constellation resolution; numeric AppView IDs are not accepted")
	addBodyFlags(cmd, &flags)
	return cmd
}

func issueDependencies(cmd *cobra.Command) (*config.Config, *tangled.IssueService, string, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, "", err
	}
	context, err := currentRepoContext(cmd, cfg)
	if err != nil {
		return nil, nil, "", err
	}
	repoURI, err := tangled.BuildRepoATURI(cmd.Context(), context)
	if err != nil {
		return nil, nil, "", err
	}
	return cfg, tangled.NewIssueService(cfg, nil), repoURI, nil
}

func currentRepoContext(cmd *cobra.Command, cfg *config.Config) (*repo.RepositoryContext, error) {
	if selector := repoFlagValue(cmd); selector != "" {
		return repo.ResolveSelector(selector, cfg)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	context, err := repo.Resolve(cmd.Context(), cwd, cfg)
	if err != nil {
		return nil, err
	}
	return context, nil
}

func repoFlagValue(cmd *cobra.Command) string {
	flag := cmd.Flag("repo")
	if flag == nil {
		return ""
	}
	return strings.TrimSpace(flag.Value.String())
}

func addBodyFlags(cmd *cobra.Command, flags *bodyFlags) {
	cmd.Flags().StringVar(&flags.body, "body", "", "Body text")
	cmd.Flags().StringVarP(&flags.bodyFile, "body-file", "F", "", "Read body from file, or '-' for stdin")
}

func resolveIssueArg(cmd *cobra.Command, service *tangled.IssueService, repoURI, input string, atproto bool) (tangled.Issue, error) {
	normalized := strings.TrimPrefix(input, "#")
	if number, parseErr := strconv.Atoi(normalized); parseErr == nil {
		if atproto {
			return tangled.Issue{}, fmt.Errorf("numeric issue IDs are AppView IDs; use an issue rkey or AT URI with --atproto")
		}
		cfg, err := config.Load()
		if err != nil {
			return tangled.Issue{}, err
		}
		context, err := currentRepoContext(cmd, cfg)
		if err != nil {
			return tangled.Issue{}, err
		}
		issue, err := service.ResolveIssueNumber(cmd.Context(), cfg.AppView.URL, context.Owner, context.Name, number)
		if err != nil {
			return tangled.Issue{}, err
		}
		return *issue, nil
	}
	if strings.HasPrefix(input, "at://") {
		issue, err := service.GetIssue(cmd.Context(), input)
		if err != nil {
			return tangled.Issue{}, err
		}
		return *issue, nil
	}
	issues, err := service.ListIssues(cmd.Context(), repoURI, tangled.IssueListOptions{State: "all", Limit: 100})
	if err == nil {
		if issue, resolveErr := tangled.ResolveIssueIdentifier(input, issues); resolveErr == nil {
			return issue, nil
		}
	}
	session, loadErr := auth.Load()
	if loadErr != nil {
		if err != nil {
			return tangled.Issue{}, err
		}
		return tangled.Issue{}, loadErr
	}
	issue, err := service.GetIssue(cmd.Context(), fmt.Sprintf("at://%s/sh.tangled.repo.issue/%s", session.DID, strings.TrimPrefix(input, "#")))
	if err != nil {
		return tangled.Issue{}, err
	}
	return *issue, nil
}

func issueDisplayID(issue tangled.Issue) string {
	if issue.Number > 0 {
		return fmt.Sprintf("#%d", issue.Number)
	}
	if issue.RKey != "" {
		return issue.RKey
	}
	if rkey := tangled.RKeyFromURI(issue.URI); rkey != "" {
		return rkey
	}
	return issue.URI
}
