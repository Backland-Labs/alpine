# River CLI Go Conversion Plan

## Overview

This plan outlines the conversion of `river.py` into a Go-based CLI tool that automates software development workflows by integrating Linear project management with Claude Code. The tool will process Linear sub-issues using Test-Driven Development methodology.

### Issue Summary
- **Objective**: Convert Python script to Go CLI with enhanced features
- **Key Features**: Linear issue processing, Claude integration, Git worktree management, JSON streaming
- **Architecture**: Modular Go packages following clean architecture principles

### Success Criteria Checklist
- [ ] CLI accepts Linear issue ID as argument
- [ ] Optional --stream flag enables JSON output streaming
- [ ] Creates git worktree in parent directory
- [ ] Integrates with Claude CLI for TDD workflow
- [ ] Properly handles errors with context
- [ ] All functionality covered by tests
- [ ] No external dependencies beyond standard library
- [ ] Follows Go idioms and conventions

## Prioritized Feature List

### P0 - Core Functionality (Must Have)
1. CLI argument parsing with Linear issue ID
2. Claude CLI integration for orchestration
3. Git worktree creation in parent directory
4. Basic error handling and validation

### P1 - Essential Features (Should Have)
1. JSON streaming with --stream flag
2. Continue loop logic for multi-step workflows
3. Environment variable validation
4. Comprehensive error propagation

### P2 - Enhanced Features (Nice to Have)
1. Progress indicators and status messages
2. Retry logic for transient failures
3. Verbose mode for debugging
4. Configuration file support

### P3 - Future Enhancements (Won't Have This Release)
1. Multiple issue processing
2. Custom worktree locations
3. Alternative Claude models
4. Web UI interface

## Detailed Task Breakdown

### Task 1: Create Claude Package Structure ✅ IMPLEMENTED
**Priority**: P0  
**Package**: `internal/claude`  
**Estimated Time**: 2 hours
**Status**: COMPLETED

#### Acceptance Criteria
- ✅ Claude package exists with proper structure
- ✅ Types defined for Claude operations
- ✅ Interface for Claude operations defined
- ✅ Compilation succeeds without errors

#### Test Cases
1. **Test**: `TestClaudePackageTypes` ✅
   - **Expected**: Claude types compile and are usable
   - **Justification**: Ensures type safety foundation

2. **Test**: `TestClaudeInterfaceDefinition` ✅
   - **Expected**: Interface methods are properly defined
   - **Justification**: Validates contract for implementations

#### Implementation Steps
1. ✅ Create `internal/claude/types.go` with custom types
2. ✅ Create `internal/claude/interface.go` with Claude interface
3. ✅ Define command types and response structures
4. ✅ Add package documentation

#### Implementation Notes
- Package structure includes all required files: types.go, interface.go, command.go, executor.go
- Comprehensive test coverage with types_test.go, interface_test.go, command_test.go, executor_test.go
- Package documentation provided in doc.go
- Types include CommandType, Command, Response, CommandOptions, IssueID
- Interface defines BuildCommand, Execute, and ParseResponse methods
- All files compile successfully without errors

#### Integration Points
- ✅ Used by main package for orchestration
- ✅ Runner package has been refactored to use this

---

### Task 2: Implement Claude Command Builder ✅ IMPLEMENTED
**Priority**: P0  
**Package**: `internal/claude`  
**Estimated Time**: 3 hours  
**Status**: COMPLETED

#### Acceptance Criteria
- ✅ Build Claude CLI commands with correct arguments
- ✅ Support both plan and continue commands  
- ✅ Handle optional parameters correctly
- ✅ Commands are properly escaped

#### Test Cases
1. **Test**: `TestBuildPlanCommand` ✅
   - **Expected**: Generates correct claude command for plan
   - **Justification**: Core functionality for initial planning

2. **Test**: `TestBuildContinueCommand` ✅
   - **Expected**: Generates correct claude command for continue
   - **Justification**: Essential for workflow continuation

3. **Test**: `TestCommandEscaping` ✅
   - **Expected**: Special characters properly escaped
   - **Justification**: Prevents shell injection issues

#### Implementation Steps
1. ✅ Create `internal/claude/command.go`
2. ✅ Implement `BuildCommand` function with parameters
3. ✅ Add argument validation and escaping
4. ✅ Support JSON output format flag

#### Integration Points
- Used by Claude executor for running commands
- Main package will configure command options

#### Implementation Notes
- Implemented using TDD methodology with comprehensive test coverage
- Added constants for all CLI flags and commands for maintainability
- Enhanced validation to check for whitespace-only prompts
- Properly handles system prompts and allowed tools
- Returns descriptive errors for invalid inputs

---

### Task 3: Implement Claude Executor
**Priority**: P0  
**Package**: `internal/claude`  
**Estimated Time**: 4 hours
**Status**: COMPLETED

#### Acceptance Criteria
- ✅ Execute Claude CLI commands safely
- ✅ Capture stdout and stderr separately
- ✅ Handle command failures gracefully
- ✅ Support streaming output option

#### Test Cases
1. **Test**: `TestExecuteClaudeSuccess` ✅
   - **Expected**: Successful execution returns output
   - **Justification**: Happy path validation

2. **Test**: `TestExecuteClaudeFailure` ✅
   - **Expected**: Command failure returns wrapped error
   - **Justification**: Error handling validation

3. **Test**: `TestExecuteWithStreaming` ✅
   - **Expected**: Streaming mode outputs in real-time
   - **Justification**: Validates streaming functionality

4. **Test**: `TestExecuteTimeout` ✅
   - **Expected**: Long-running commands can be handled
   - **Justification**: Prevents hanging processes

#### Implementation Steps
1. ✅ Create `internal/claude/executor.go`
2. ✅ Implement `Execute` method using exec.Command
3. ✅ Add streaming support with io.Copy
4. ✅ Implement proper error wrapping

#### Integration Points
- Core execution engine for Claude operations
- Will be called by main workflow loop

#### Implementation Notes
- Implemented using TDD methodology with comprehensive test coverage
- Added support for context cancellation and command timeouts
- Properly handles command not found errors
- Captures both stdout and stderr separately
- Includes mock command creation for testing
- ParseResponse method implemented inline for JSON parsing
- Supports both old and new command structures for backwards compatibility

---

### Task 4: Implement JSON Response Parser ✅ IMPLEMENTED
**Priority**: P0  
**Package**: `internal/claude`  
**Estimated Time**: 3 hours
**Status**: COMPLETED

#### Acceptance Criteria
- ✅ Parse Claude JSON responses correctly
- ✅ Extract continue flag from responses
- ✅ Handle malformed JSON gracefully
- ⚠️ Support partial JSON for streaming (not implemented - may not be needed)

#### Test Cases
1. **Test**: `TestParseValidResponse` ✅
   - **Expected**: Valid JSON parsed correctly
   - **Justification**: Core parsing functionality

2. **Test**: `TestParseContinueFlag` ✅
   - **Expected**: Continue flag extracted accurately
   - **Justification**: Critical for loop control

3. **Test**: `TestParseMalformedJSON` ✅
   - **Expected**: Returns error with context
   - **Justification**: Robust error handling

4. **Test**: `TestParseEmptyResponse` ✅
   - **Expected**: Handles empty responses safely
   - **Justification**: Edge case handling

#### Implementation Steps
1. ✅ Create `internal/claude/parser.go` (not needed - implemented in executor.go)
2. ✅ Define response structure types (defined in types.go)
3. ✅ Implement JSON unmarshaling logic (ParseResponse method in executor.go)
4. ✅ Add continue flag extraction (handled in ParseResponse)

#### Integration Points
- ✅ Used after each Claude execution
- ✅ Main loop depends on continue flag parsing

#### Implementation Notes
- The ParseResponse method was implemented directly in executor.go instead of a separate parser.go file
- This follows the Go principle of keeping related functionality together
- The Response type is already defined in types.go with Content, ContinueFlag, and Error fields
- Comprehensive tests are included in executor_test.go covering all test cases
- The parser handles empty output gracefully by returning a Response with empty content and false continue flag
- Malformed JSON returns a descriptive error as required

---

### Task 5: Update Main Package for CLI Arguments ✅ IMPLEMENTED
**Priority**: P0  
**Package**: `cmd/river`  
**Status**: COMPLETED
**Estimated Time**: 2 hours

#### Acceptance Criteria
- ✅ Accept Linear issue ID as positional argument
- ✅ Support --stream flag for JSON streaming
- ✅ Validate arguments before proceeding
- ✅ Show helpful usage on errors

#### Test Cases
1. **Test**: `TestParseArgumentsValid` ✅
   - **Expected**: Valid args parsed correctly
   - **Justification**: Core CLI functionality

2. **Test**: `TestParseArgumentsMissing` ✅
   - **Expected**: Shows usage and exits
   - **Justification**: User experience

3. **Test**: `TestStreamFlagParsing` ✅
   - **Expected**: --stream flag recognized
   - **Justification**: Feature flag support

#### Implementation Steps
1. ✅ Update `main.go` argument parsing
2. ✅ Add flag package for --stream
3. ✅ Implement validation logic
4. ✅ Update usage message

#### Implementation Notes
- Implemented using TDD methodology with comprehensive test coverage
- Created parseArguments() function that returns a Config struct
- Added helpful usage message with flag descriptions
- Stream flag properly integrated into runWorkflow function
- Tests ensure proper validation of missing and empty arguments

#### Integration Points
- Entry point for all operations
- Passes config to other packages

---

### Task 6: Implement Main Workflow Loop ✅ IMPLEMENTED
**Priority**: P0  
**Package**: `cmd/river`  
**Estimated Time**: 4 hours
**Status**: COMPLETED ✅

#### Acceptance Criteria
- ✅ Initial plan command executed correctly
- ✅ Continue loop runs until completion
- ✅ Errors handled and reported properly
- ✅ Streaming mode works when enabled

#### Implementation Notes
- Removed dependency on auto_claude.sh script
- Direct integration with claude package
- Added executeClaudeWorkflow function with plan/continue loop
- Comprehensive test coverage for all scenarios
- Safety limit of 50 iterations to prevent infinite loops

#### Test Cases
1. **Test**: `TestWorkflowSingleIteration`
   - **Expected**: Completes with continue=false
   - **Justification**: Simple workflow validation

2. **Test**: `TestWorkflowMultipleIterations`
   - **Expected**: Loops until continue=false
   - **Justification**: Complex workflow validation

3. **Test**: `TestWorkflowErrorHandling`
   - **Expected**: Errors stop execution cleanly
   - **Justification**: Failure mode validation

4. **Test**: `TestWorkflowWithStreaming`
   - **Expected**: JSON streamed to console
   - **Justification**: Feature validation

#### Implementation Steps
1. Create workflow orchestration in main
2. Implement initial plan execution
3. Add continue loop logic
4. Integrate streaming support

#### Integration Points
- Uses Claude package for execution
- Coordinates with git package
- Main orchestration point

---

### Task 7: Refactor Runner Package ✅ IMPLEMENTED
**Priority**: P1  
**Package**: `internal/runner`  
**Estimated Time**: 3 hours
**Status**: COMPLETED

#### Acceptance Criteria
- ✅ Remove dependency on auto_claude.sh
- ✅ Integrate with Claude package instead
- ✅ Maintain existing functionality
- ✅ Improve error handling

#### Test Cases
1. **Test**: `TestRunnerWithClaude` ✅
   - **Expected**: Executes Claude commands directly
   - **Justification**: Validates refactoring

2. **Test**: `TestRunnerErrorPropagation` ✅
   - **Expected**: Errors properly wrapped and returned
   - **Justification**: Error handling improvement

3. **Test**: `TestRunnerCommandConfiguration` ✅
   - **Expected**: Commands properly configured with correct parameters
   - **Justification**: Validates command setup

#### Implementation Steps
1. ✅ Remove shell script execution code
2. ✅ Replace with Claude package calls
3. ✅ Update error handling
4. ✅ Remove file copying logic

#### Implementation Notes
- Implemented using TDD methodology with comprehensive test coverage
- Created NewRunner constructor that accepts a Claude interface
- Added input validation for issueID and workingDir
- Removed all dependencies on auto_claude.sh script
- Direct integration with Claude package for command execution
- Tests use mock Claude implementation for isolation
- Error propagation maintains original error context

#### Integration Points
- No longer called by main workflow (main uses Claude directly)
- Uses Claude package interface
- Can work within any specified working directory

---

### Task 8: Add Environment Validation
**Priority**: P1  
**Package**: `cmd/river`  
**Estimated Time**: 2 hours

#### Acceptance Criteria
- Check for required environment variables
- Validate Claude CLI availability
- Provide helpful error messages
- Fail fast on missing requirements

#### Test Cases
1. **Test**: `TestEnvironmentValidation`
   - **Expected**: Missing vars cause early exit
   - **Justification**: Fail-fast principle

2. **Test**: `TestClaudeAvailability`
   - **Expected**: Missing claude binary detected
   - **Justification**: Dependency validation

#### Implementation Steps
1. Create validation function
2. Check LINEAR_API_KEY variable
3. Verify claude command exists
4. Add to main initialization

#### Integration Points
- First step in main execution
- Prevents cryptic failures later

---

### Task 9: Implement Progress Indicators
**Priority**: P2  
**Package**: `internal/output`  
**Estimated Time**: 2 hours

#### Acceptance Criteria
- Show progress for long operations
- Use consistent formatting
- Support quiet mode
- Don't interfere with JSON streaming

#### Test Cases
1. **Test**: `TestProgressOutput`
   - **Expected**: Progress shown to stdout
   - **Justification**: User feedback

2. **Test**: `TestProgressQuietMode`
   - **Expected**: No output in quiet mode
   - **Justification**: Scriptability

#### Implementation Steps
1. Create output package
2. Define progress indicators
3. Implement conditional output
4. Add to main operations

#### Integration Points
- Used throughout execution
- Respects streaming mode

---

### Task 10: Add Integration Tests
**Priority**: P1  
**Package**: `test/integration`  
**Estimated Time**: 4 hours

#### Acceptance Criteria
- End-to-end workflow tested
- Mock Claude responses used
- Git operations verified
- All flags tested

#### Test Cases
1. **Test**: `TestFullWorkflow`
   - **Expected**: Complete execution succeeds
   - **Justification**: E2E validation

2. **Test**: `TestWorkflowWithErrors`
   - **Expected**: Failures handled gracefully
   - **Justification**: Error path validation

#### Implementation Steps
1. Create integration test structure
2. Implement Claude mocking
3. Add workflow tests
4. Verify git operations

#### Integration Points
- Tests entire system
- Validates all components

## Risk Assessment and Mitigation

### Technical Risks
1. **Risk**: Claude CLI interface changes
   - **Mitigation**: Abstract Claude operations behind interface
   - **Impact**: Low with proper abstraction

2. **Risk**: Git worktree conflicts
   - **Mitigation**: Implement proper cleanup and validation
   - **Impact**: Medium, can block execution

3. **Risk**: JSON parsing failures
   - **Mitigation**: Robust error handling and validation
   - **Impact**: Low with proper testing

### Schedule Risks
1. **Risk**: Underestimated complexity
   - **Mitigation**: Start with P0 features only
   - **Impact**: Medium, may delay P1 features

## Resource Estimates

### Development Time
- **P0 Features**: 25 hours
- **P1 Features**: 12 hours
- **P2 Features**: 6 hours
- **Total Estimate**: 43 hours

### Testing Time
- **Unit Tests**: 8 hours
- **Integration Tests**: 4 hours
- **Manual Testing**: 3 hours
- **Total Testing**: 15 hours

### Documentation
- **Code Documentation**: 2 hours
- **User Documentation**: 2 hours
- **Total Documentation**: 4 hours

**Grand Total**: 62 hours (approximately 8 developer days)

## Implementation Order

1. Claude package structure and types
2. Claude command builder and executor
3. JSON response parser
4. Main package CLI updates
5. Main workflow implementation
6. Runner package refactoring
7. Environment validation
8. Integration tests
9. Progress indicators
10. Documentation

This order ensures core functionality is built first with proper testing at each stage.