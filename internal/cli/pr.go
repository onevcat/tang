package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	core "tangled.org/core/api/tangled"
	"tangled.org/onev.cat/tang/internal/atproto"
	"tangled.org/onev.cat/tang/internal/auth"
	"tangled.org/onev.cat/tang/internal/config"
	tanggit "tangled.org/onev.cat/tang/internal/git"
	localrepo "tangled.org/onev.cat/tang/internal/repo"
	"tangled.org/onev.cat/tang/internal/tangled"
)

var (
	resolveDIDForCLI    = atproto.ResolveDID
	resolveHandleForCLI = atproto.ResolveHandle
)

func newPRCommand(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "pr", Short: "Manage Tangled pull requests"}
	cmd.AddCommand(newPRListCommand(opts))
	cmd.AddCommand(newPRViewCommand(opts))
	cmd.AddCommand(newPRCreateCommand(opts))
	cmd.AddCommand(newPRStateCommand(opts, "close", "closed"))
	cmd.AddCommand(newPRStateCommand(opts, "reopen", "open"))
	cmd.AddCommand(newPRDiffCommand())
	cmd.AddCommand(newPRCommentCommand(opts))
	cmd.AddCommand(newPRCheckoutCommand())
	cmd.AddCommand(newPRMergeCommand())
	return cmd
}

func newPRListCommand(opts *RootOptions) *cobra.Command {
	var state string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pull requests",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, service, repoURI, _, err := prDependencies(cmd)
			if err != nil {
				return err
			}
			pulls, err := service.ListPulls(cmd.Context(), repoURI, state, 50)
			if err != nil {
				return err
			}
			if rendered, err := renderJSONIfRequested(cmd, opts, pulls); rendered || err != nil {
				return err
			}
			for _, pull := range pulls {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "#%d\t%s\t%s\t%s\n", pull.Number, pull.Title, pull.Status, pull.Author); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&state, "state", "open", "Filter by state: open, closed, merged, all")
	return cmd
}

func newPRViewCommand(opts *RootOptions) *cobra.Command {
	var web bool
	cmd := &cobra.Command{
		Use:   "view <pull>",
		Short: "View a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, service, repoURI, context, err := prDependencies(cmd)
			if err != nil {
				return err
			}
			pull, err := resolvePullArg(cmd, service, repoURI, args[0])
			if err != nil {
				return err
			}
			if web {
				return openBrowserForCLI(strings.TrimRight(cfg.AppView.URL, "/") + "/" + context.Owner + "/" + context.Name + "/pulls/" + fmt.Sprint(pull.Number))
			}
			if repoRecord, err := tangled.NewRepoService(cfg, nil).GetRepo(cmd.Context(), context.Owner, context.Name); err == nil {
				if ownerDID, _, err := resolveRepoOwnerForCLI(cmd, context.Owner); err == nil {
					if mergeable, err := service.MergeCheck(cmd.Context(), *repoRecord, ownerDID, pull); err == nil {
						pull.Mergeable = mergeable
					} else {
						pull.Mergeable = "unknown: " + err.Error()
					}
				}
			}
			if rendered, err := renderJSONIfRequested(cmd, opts, pull); rendered || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Pull #%d %s\nTitle: %s\nAuthor: %s\nTarget: %s\nSource: %s\nMerge: %s\nURI: %s\n\n%s\n", pull.Number, pull.Status, pull.Title, pull.Author, pull.Target, pull.Branch, pull.Mergeable, pull.URI, pull.Body)
			return err
		},
	}
	cmd.Flags().BoolVar(&web, "web", false, "Open the pull request in a browser")
	return cmd
}

func newPRCreateCommand(opts *RootOptions) *cobra.Command {
	var title, base, head string
	var fill bool
	flags := bodyFlags{}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a pull request",
		RunE: func(cmd *cobra.Command, _ []string) error {
			session, err := auth.Load()
			if err != nil {
				return err
			}
			_, service, repoURI, context, err := prDependencies(cmd)
			if err != nil {
				return err
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			repoRecord, err := tangled.NewRepoService(cfg, nil).GetRepo(cmd.Context(), context.Owner, context.Name)
			if err != nil {
				return err
			}
			if head == "" {
				cwd, _ := os.Getwd()
				head, err = tanggit.CurrentBranch(cmd.Context(), cwd, tanggit.GitRunner{})
				if err != nil {
					return err
				}
			}
			body, _, err := readBodyInput(flags, cmd.InOrStdin())
			if err != nil {
				return err
			}
			pull, err := service.CreatePull(cmd.Context(), session, tangled.PullCreateOptions{
				Repo:       *repoRecord,
				RepoURI:    repoURI,
				BaseBranch: base,
				HeadBranch: head,
				Title:      title,
				Body:       body,
				Fill:       fill,
			})
			if err != nil {
				return err
			}
			if rendered, err := renderJSONIfRequested(cmd, opts, pull); rendered || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created pull %s\n", tangled.RKeyFromURI(pull.URI))
			return err
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Pull request title")
	cmd.Flags().StringVar(&base, "base", "main", "Base branch")
	cmd.Flags().StringVar(&head, "head", "", "Head branch")
	cmd.Flags().BoolVar(&fill, "fill", false, "Fill title from patch")
	addBodyFlags(cmd, &flags)
	return cmd
}

func newPRStateCommand(opts *RootOptions, name, state string) *cobra.Command {
	return &cobra.Command{
		Use:   name + " <pull>",
		Short: name + " a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := auth.Load()
			if err != nil {
				return err
			}
			_, service, repoURI, _, err := prDependencies(cmd)
			if err != nil {
				return err
			}
			pull, err := resolvePullArg(cmd, service, repoURI, args[0])
			if err != nil {
				return err
			}
			if err := service.SetPullStatus(cmd.Context(), session, pull.URI, state); err != nil {
				return err
			}
			pull.Status = state
			if rendered, err := renderJSONIfRequested(cmd, opts, pull); rendered || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Pull #%d is now %s\n", pull.Number, state)
			return err
		},
	}
}

func newPRDiffCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <pull>",
		Short: "Print pull request patch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, service, repoURI, _, err := prDependencies(cmd)
			if err != nil {
				return err
			}
			pull, err := resolvePullArg(cmd, service, repoURI, args[0])
			if err != nil {
				return err
			}
			patch, err := service.FetchPullPatch(cmd.Context(), pull.URI)
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), patch)
			return err
		},
	}
}

func newPRCommentCommand(opts *RootOptions) *cobra.Command {
	flags := bodyFlags{}
	cmd := &cobra.Command{
		Use:   "comment <pull>",
		Short: "Comment on a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := auth.Load()
			if err != nil {
				return err
			}
			_, service, repoURI, _, err := prDependencies(cmd)
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
			pull, err := resolvePullArg(cmd, service, repoURI, args[0])
			if err != nil {
				return err
			}
			comment, err := service.AddPullComment(cmd.Context(), session, pull.URI, body)
			if err != nil {
				return err
			}
			if rendered, err := renderJSONIfRequested(cmd, opts, comment); rendered || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Commented on pull #%d\n", pull.Number)
			return err
		},
	}
	addBodyFlags(cmd, &flags)
	return cmd
}

func newPRCheckoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "checkout <pull>",
		Short: "Checkout a pull request source branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, service, repoURI, context, err := prDependencies(cmd)
			if err != nil {
				return err
			}
			pull, err := resolvePullArg(cmd, service, repoURI, args[0])
			if err != nil {
				return err
			}
			if pull.Branch == "" {
				return tangled.ErrPatchOnlyCheckout
			}
			cwd, _ := os.Getwd()
			return tanggit.CheckoutBranchFromRemote(cmd.Context(), cwd, context.RemoteName, pull.Branch)
		},
	}
}

func newPRMergeCommand() *cobra.Command {
	var subject, body string
	cmd := &cobra.Command{
		Use:   "merge <pull>",
		Short: "Merge a pull request patch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := auth.Load()
			if err != nil {
				return err
			}
			cfg, service, repoURI, context, err := prDependencies(cmd)
			if err != nil {
				return err
			}
			pull, err := resolvePullArg(cmd, service, repoURI, args[0])
			if err != nil {
				return err
			}
			patch, err := service.FetchPullPatch(cmd.Context(), pull.URI)
			if err != nil {
				return err
			}
			repoRecord, err := tangled.NewRepoService(cfg, nil).GetRepo(cmd.Context(), context.Owner, context.Name)
			if err != nil {
				return err
			}
			ownerDID, _, err := resolveRepoOwnerForCLI(cmd, context.Owner)
			if err != nil {
				return err
			}
			token, err := tangled.NewPDSClient(session, nil).GetServiceAuth(cmd.Context(), repoRecord.Knot, core.RepoMergeNSID, 20*time.Minute)
			if err != nil {
				return err
			}
			message := subject
			if message == "" {
				message = pull.Title
			}
			commitBody := body
			if commitBody == "" {
				commitBody = pull.Body
			}
			author := session.Handle
			if err := tangled.NewKnotClient(repoRecord.Knot, tangled.WithServiceAuthToken(token)).Merge(cmd.Context(), &core.RepoMerge_Input{
				Did:           ownerDID,
				Name:          repoRecord.Name,
				Branch:        pull.Target,
				Patch:         patch,
				CommitMessage: &message,
				CommitBody:    optionalCLIString(commitBody),
				AuthorName:    &author,
			}); err != nil {
				return err
			}
			_ = service.SetPullStatus(cmd.Context(), session, pull.URI, "merged")
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Merged pull %s\n", tangled.RKeyFromURI(pull.URI))
			return err
		},
	}
	cmd.Flags().StringVar(&subject, "subject", "", "Merge commit subject")
	cmd.Flags().StringVar(&body, "body", "", "Merge commit body")
	return cmd
}

func prDependencies(cmd *cobra.Command) (*config.Config, *tangled.PullService, string, *localrepo.RepositoryContext, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, "", nil, err
	}
	context, err := currentRepoContext(cmd, cfg)
	if err != nil {
		return nil, nil, "", nil, err
	}
	repoURI, err := tangled.BuildRepoATURI(cmd.Context(), context)
	if err != nil {
		return nil, nil, "", nil, err
	}
	return cfg, tangled.NewPullService(cfg, nil), repoURI, context, nil
}

func resolvePullArg(cmd *cobra.Command, service *tangled.PullService, repoURI, input string) (tangled.Pull, error) {
	pulls, err := service.ListPulls(cmd.Context(), repoURI, "all", 100)
	if err == nil {
		if pull, resolveErr := tangled.ResolvePullIdentifier(input, pulls); resolveErr == nil {
			return pull, nil
		}
	}
	if _, parseErr := strconv.Atoi(strings.TrimPrefix(input, "#")); parseErr == nil {
		return tangled.Pull{}, fmt.Errorf("pull %s not found", input)
	}
	session, loadErr := auth.Load()
	if loadErr != nil {
		if err != nil {
			return tangled.Pull{}, err
		}
		return tangled.Pull{}, loadErr
	}
	pull, err := service.GetPull(cmd.Context(), fmt.Sprintf("at://%s/sh.tangled.repo.pull/%s", session.DID, strings.TrimPrefix(input, "#")))
	if err != nil {
		return tangled.Pull{}, err
	}
	return *pull, nil
}

func resolveRepoOwnerForCLI(cmd *cobra.Command, owner string) (string, string, error) {
	return tangledResolveOwner(cmd, owner)
}

func tangledResolveOwner(cmd *cobra.Command, owner string) (string, string, error) {
	if strings.HasPrefix(owner, "did:") {
		ident, err := resolveDIDForCLI(cmd.Context(), owner)
		if err != nil {
			return "", "", err
		}
		return owner, ident.PDS, nil
	}
	ident, err := resolveHandleForCLI(cmd.Context(), owner)
	if err != nil {
		return "", "", err
	}
	return ident.DID, ident.PDS, nil
}

func optionalCLIString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
