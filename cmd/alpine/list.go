package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// composeLsEntry represents a project from "docker compose ls --format json".
// This is distinct from composeProject in docker.go which parses
// "docker compose ps" output (per-container with Service field).
type composeLsEntry struct {
	Name        string `json:"Name"`
	Status      string `json:"Status"`
	ConfigFiles string `json:"ConfigFiles"`
}

// containerInfo represents a container from docker ps --format json.
type containerInfo struct {
	Names  string `json:"Names"`
	Labels string `json:"Labels"`
	State  string `json:"State"`
}

// envInfo is the output format for the list command.
type envInfo struct {
	Name    string `json:"name"`
	Branch  string `json:"branch"`
	Status  string `json:"status"`
	Created string `json:"created"`
}

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "Show all active environments",
	Aliases: []string{"ls"},
	RunE:    runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

// parseContainerLabels splits a Docker label string (comma-separated key=value pairs)
// and returns a map of label keys to values.
func parseContainerLabels(labels string) map[string]string {
	result := make(map[string]string)
	if labels == "" {
		return result
	}
	for _, pair := range strings.Split(labels, ",") {
		k, v, ok := strings.Cut(pair, "=")
		if ok {
			result[k] = v
		}
	}
	return result
}

// parseNDJSON parses newline-delimited JSON (one JSON object per line) into a slice
// of containerInfo structs.
func parseNDJSON(data string) []containerInfo {
	var containers []containerInfo
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var c containerInfo
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			slog.Debug("skipping unparseable container line", "line", line, "error", err)
			continue
		}
		containers = append(containers, c)
	}
	return containers
}

// buildEnvList correlates compose projects with container metadata to produce
// the final list of environment info for display.
func buildEnvList(projects []composeLsEntry, containers []containerInfo) []envInfo {
	// Build a lookup: project name -> container metadata.
	// Container names from docker ps contain the project prefix (e.g. alpine-myenv-dev-1).
	// We match containers to projects by checking if the container name starts with
	// the project name.
	type metadata struct {
		branch  string
		created string
	}
	projectMeta := make(map[string]metadata)

	for _, c := range containers {
		labels := parseContainerLabels(c.Labels)
		name := labels["alpine.name"]
		if name == "" {
			// Try matching by container name prefix.
			for _, p := range projects {
				if strings.HasPrefix(c.Names, p.Name) {
					name = strings.TrimPrefix(p.Name, "alpine-")
					break
				}
			}
		}
		if name != "" {
			projectMeta[name] = metadata{
				branch:  labels["alpine.branch"],
				created: labels["alpine.created"],
			}
		}
	}

	envs := make([]envInfo, 0, len(projects))
	for _, p := range projects {
		displayName := strings.TrimPrefix(p.Name, "alpine-")

		// Extract a short status from the compose project status string.
		// docker compose ls returns statuses like "running(2)" or "exited(1), running(1)".
		status := normalizeStatus(p.Status)

		meta, ok := projectMeta[displayName]
		branch := ""
		created := ""
		if ok {
			branch = meta.branch
			created = meta.created
		}

		envs = append(envs, envInfo{
			Name:    displayName,
			Branch:  branch,
			Status:  status,
			Created: created,
		})
	}

	return envs
}

// normalizeStatus simplifies a docker compose project status string for display.
// Input examples: "running(2)", "exited(1), running(1)", "created(1)"
// Output: "running", "partial", "stopped"
func normalizeStatus(raw string) string {
	lower := strings.ToLower(raw)
	switch {
	case strings.Contains(lower, "running") && strings.Contains(lower, "exited"):
		return "partial"
	case strings.Contains(lower, "running"):
		return "running"
	case strings.Contains(lower, "exited"), strings.Contains(lower, "dead"):
		return "stopped"
	case strings.Contains(lower, "created"):
		return "created"
	default:
		return raw
	}
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	if err := dockerHealthCheck(ctx); err != nil {
		return err
	}

	// Get compose projects filtered by the alpine- prefix.
	stdout, _, err := run(ctx, "docker", "compose", "ls", "--filter", "name=alpine-", "--format", "json")
	if err != nil {
		return fmt.Errorf("listing environments: %w", err)
	}

	var projects []composeLsEntry
	trimmed := strings.TrimSpace(stdout)
	if trimmed != "" {
		if err := json.Unmarshal([]byte(trimmed), &projects); err != nil {
			return fmt.Errorf("parsing compose output: %w", err)
		}
	}

	if len(projects) == 0 {
		if jsonOutput {
			return outputJSON([]envInfo{})
		}
		fmt.Println("No active environments. Run 'alpine create <name>' to get started.")
		return nil
	}

	// Get container metadata (labels) for alpine-managed containers.
	// docker ps --format json outputs NDJSON (one JSON object per line).
	containerStdout, _, err := run(ctx, "docker", "ps",
		"--filter", "label=alpine.managed=true",
		"--format", "json",
	)
	var containers []containerInfo
	if err != nil {
		// Non-fatal: we can still show projects without metadata.
		slog.Debug("failed to get container metadata", "error", err)
	} else {
		containers = parseNDJSON(containerStdout)
	}

	envs := buildEnvList(projects, containers)

	if jsonOutput {
		return outputJSON(envs)
	}

	// Table output.
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tBRANCH\tSTATUS\tCREATED") //nolint:errcheck // errors surface on Flush
	for _, env := range envs {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", env.Name, env.Branch, env.Status, env.Created) //nolint:errcheck // errors surface on Flush
	}
	return w.Flush()
}
