package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	launchRepo          string
	launchImageProfile  string
	launchTask          string
	launchForceRecreate bool
)

type launchOutput struct {
	Name           string         `json:"name"`
	State          lifecycleState `json:"state"`
	Repo           string         `json:"repo"`
	ImageProfile   string         `json:"image_profile"`
	ContainerClass string         `json:"container_class"`
	WebURL         string         `json:"web_url"`
	Reused         bool           `json:"reused"`
	Resumed        bool           `json:"resumed"`
	TaskAccepted   bool           `json:"task_accepted"`
	OperationID    string         `json:"operation_id"`
}

var launchCmd = &cobra.Command{
	Use:   "launch <name>",
	Short: "Launch or resume a cloud sandbox",
	Args:  cobra.ExactArgs(1),
	RunE:  runLaunch,
}

func init() {
	launchCmd.Flags().StringVar(&launchRepo, "repo", "", "repository URL override")
	launchCmd.Flags().StringVar(&launchImageProfile, "image-profile", "", "image profile override")
	launchCmd.Flags().StringVar(&launchTask, "task", "", "optional initial task kickoff")
	launchCmd.Flags().BoolVar(&launchForceRecreate, "force-recreate", false, "recreate when immutable identity differs")
	rootCmd.AddCommand(launchCmd)
}

func runLaunch(cmd *cobra.Command, args []string) error {
	name := sanitizeName(args[0])
	if name == "" {
		return userErr(fmt.Sprintf("invalid name %q: nothing left after sanitization", args[0]))
	}
	if err := validateName(name); err != nil {
		return userErr(err.Error())
	}

	cfg, err := loadConfig()
	if err != nil {
		return userErr(fmt.Sprintf("failed to load config: %v", err))
	}

	repo, err := cfg.resolveRepo(launchRepo)
	if err != nil {
		return userErrReason(err.Error(), "repo_resolution_failed")
	}
	profile, containerClass, err := cfg.resolveImageProfile(launchImageProfile)
	if err != nil {
		return userErrReason(err.Error(), "image_profile_unknown")
	}

	orch := newOrchestrator(cfg)
	identity, exists, err := orch.sandboxIdentity(name)
	if err != nil {
		return sysErr(fmt.Sprintf("failed to read existing identity: %v", err))
	}
	if exists && !launchForceRecreate {
		if identity.Repo != repo || identity.ImageProfile != profile {
			return userErr(fmt.Sprintf("sandbox %q already exists with repo %q and image profile %q; rerun with --force-recreate to replace", name, identity.Repo, identity.ImageProfile))
		}
	}

	if _, _, err := run(cmd.Context(), "git", "ls-remote", "--heads", repo); err != nil {
		return sysErrReason("repository is not reachable with current auth", "repo_unreachable", false)
	}

	result, err := orch.launch(launchOptions{
		Name:          name,
		Repo:          repo,
		ImageProfile:  profile,
		Task:          launchTask,
		ForceRecreate: launchForceRecreate,
	})
	if err != nil {
		return err
	}

	out := launchOutput{
		Name:           result.Name,
		State:          result.State,
		Repo:           result.Repo,
		ImageProfile:   result.ImageProfile,
		ContainerClass: containerClass,
		WebURL:         result.WebURL,
		Reused:         result.Reused,
		Resumed:        result.Resumed,
		TaskAccepted:   launchTask != "",
		OperationID:    result.OperationID,
	}

	if jsonOutput {
		return outputJSON(out)
	}

	fmt.Printf("Sandbox %s is %s\n", out.Name, out.State)
	if out.Reused {
		fmt.Printf("Reused existing identity for repo %s (%s)\n", out.Repo, out.ImageProfile)
	} else {
		fmt.Printf("Launched sandbox for repo %s (%s)\n", out.Repo, out.ImageProfile)
	}
	if out.Resumed {
		fmt.Printf("Resumed from durable checkpoint\n")
	}
	if out.TaskAccepted {
		fmt.Printf("Initial task accepted\n")
	}
	fmt.Printf("OpenCode URL: %s\n", out.WebURL)

	return nil
}
