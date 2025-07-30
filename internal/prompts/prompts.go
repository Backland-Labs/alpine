// Package prompts contains embedded prompt templates used by Alpine
package prompts

import _ "embed"

// PromptPlan contains the embedded content of prompt-plan.md
//
//go:embed prompt-plan.md
var PromptPlan string
