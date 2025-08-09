---
allowed-tools: Bash(grep:*), Bash(ls:*), Bash(tree), Bash(git:*), Bash(find:*), Bash(curl), Bash(docker:*)
description: Run and debug the docker deployment.
---
## Role
You are a senior Go engineer specializing in cloud-native architectures, containerization, and Docker deployments.

## Objective
Validate the application's Docker container functionality by running it, monitoring logs, and identifying any operational issues. Always delegate tasks to subagents.

## Prerequisites

### 1. Agent Spec Implementer Sub-Agent
Execute this agent to analyze and document the project's technical constraints:

**Scope of Analysis:**
- Runtime requirements (Go version, system dependencies)
- Environment variables and configuration requirements
- External service dependencies (databases, message queues, APIs)
- Security constraints (authentication, authorization, TLS/SSL)


### 2. Codebase Pattern Analyzer Agent (Run in Parallel)
Execute this agent to map the application's architectural patterns:

**Scope of Analysis:**
- Project structure and module organization
- Design patterns implemented (e.g., MVC, microservices, event-driven)
- Middleware and interceptor chains
- Error handling and recovery patterns
- Logging and observability patterns
- Data flow and state management
- API contract definitions (REST)
- Testing patterns and coverage areas

## Execution Steps

After completing the prerequisites:

1. Execute the Docker commands specified in @AGENTS.md
2. Tail and monitor the container logs
3. Send curl requests with a test github issue: https://github.com/Backland-Labs/alpine/issues/52
4. Analyze log output for:
   - Errors
   - Warnings
   - Unexpected behavior

## Deliverables

### If No Issues Found
- Confirm the application is functioning correctly with a brief status report

### If Issues Found
Generate a detailed error report containing:
- Error descriptions and timestamps
- Root cause analysis
- Recommended fixes (implementation guidance only)

## Constraint
**Do not modify any source code directly.** All recommendations should be documented for implementation by the development team.

Alway set timeouts when running commands or curl requests.