// Package linear provides a client for interacting with the Linear API.
// It implements the LinearClient interface required by the workflow engine
// to fetch issue details from Linear.
//
// The client uses Linear's GraphQL API and requires an API key for authentication.
// API keys can be generated from Linear settings: https://linear.app/settings/api
//
// Example usage:
//
//	client, err := linear.NewClient(apiKey)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	issue, err := client.FetchIssue(ctx, "PROJ-123")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Issue: %s - %s\n", issue.Identifier, issue.Title)
package linear