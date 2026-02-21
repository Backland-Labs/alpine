package main

import (
	"context"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var openPrintOnly bool

type openOutput struct {
	Name   string `json:"name"`
	WebURL string `json:"web_url"`
	Opened bool   `json:"opened"`
}

var openCmd = &cobra.Command{
	Use:   "open <name>",
	Short: "Open or print the OpenCode web URL",
	Args:  cobra.ExactArgs(1),
	RunE:  runOpen,
}

func init() {
	openCmd.Flags().BoolVar(&openPrintOnly, "print", false, "print URL without opening a browser")
	rootCmd.AddCommand(openCmd)
}

func runOpen(cmd *cobra.Command, args []string) error {
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

	var webURL string
	if cfg.usesControlPlane() {
		webURL, err = getOpenURLControlPlane(cmd.Context(), cfg, name)
	} else {
		orch := newOrchestrator(cfg)
		webURL, err = orch.open(name)
	}
	if err != nil {
		return err
	}

	opened := false
	openErr := ""
	if !jsonOutput && !openPrintOnly {
		if err := openBrowserURL(webURL); err == nil {
			opened = true
		} else {
			openErr = err.Error()
		}
	}

	out := openOutput{Name: args[0], WebURL: webURL, Opened: opened}
	if jsonOutput {
		return outputJSON(out)
	}

	if opened {
		fmt.Printf("Opened %s\n", webURL)
		return nil
	}

	if openErr != "" {
		fmt.Printf("Failed to open browser: %s\n", openErr)
	}
	fmt.Println(webURL)
	return nil
}

func getOpenURLControlPlane(ctx context.Context, cfg *Config, name string) (string, error) {
	client := newControlPlaneClient(cfg.Sandbox.ControlPlaneURL)
	resp, err := client.GetSandboxOpenURL(ctx, name)
	if err != nil {
		return "", err
	}
	return resp.WebURL, nil
}

func openBrowserURL(url string) error {
	return openBrowserURLForOS(runtime.GOOS, url)
}

func openBrowserURLForOS(goos, url string) error {
	switch goos {
	case "darwin":
		_, _, err := run(context.Background(), "open", url)
		return err
	case "linux":
		_, _, err := run(context.Background(), "xdg-open", url)
		return err
	case "windows":
		_, _, err := run(context.Background(), "rundll32", "url.dll,FileProtocolHandler", url)
		return err
	default:
		return fmt.Errorf("unsupported platform")
	}
}
