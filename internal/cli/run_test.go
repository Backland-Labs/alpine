package cli

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/maxmcd/river/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock interfaces for testing

type MockConfigLoader struct {
	mock.Mock
}

func (m *MockConfigLoader) Load() (*config.Config, error) {
	args := m.Called()
	return args.Get(0).(*config.Config), args.Error(1)
}

type MockWorkflowEngine struct {
	mock.Mock
}

func (m *MockWorkflowEngine) Run(ctx context.Context, taskDescription string, generatePlan bool) error {
	args := m.Called(ctx, taskDescription, generatePlan)
	return args.Error(0)
}

type MockFileReader struct {
	mock.Mock
}

func (m *MockFileReader) ReadFile(filename string) ([]byte, error) {
	args := m.Called(filename)
	return args.Get(0).([]byte), args.Error(1)
}

// Dependencies struct is now defined in interfaces.go

// TestRunWorkflowWithTaskDescription tests the main workflow execution with command line args
func TestRunWorkflowWithTaskDescription(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		noPlan           bool
		setupMocks       func(*Dependencies)
		wantErr          bool
		expectedErrorMsg string
	}{
		{
			name:   "successful execution with task description",
			args:   []string{"Implement user authentication"},
			noPlan: false,
			setupMocks: func(deps *Dependencies) {
				cfg := &config.Config{WorkDir: "/tmp"}
				deps.ConfigLoader.(*MockConfigLoader).On("Load").Return(cfg, nil)
				deps.WorkflowEngine.(*MockWorkflowEngine).On("Run", mock.Anything, "Implement user authentication", true).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "successful execution with --no-plan flag",
			args:   []string{"Fix critical bug"},
			noPlan: true,
			setupMocks: func(deps *Dependencies) {
				cfg := &config.Config{WorkDir: "/tmp"}
				deps.ConfigLoader.(*MockConfigLoader).On("Load").Return(cfg, nil)
				deps.WorkflowEngine.(*MockWorkflowEngine).On("Run", mock.Anything, "Fix critical bug", false).Return(nil)
			},
			wantErr: false,
		},
		{
			name:             "empty task description",
			args:             []string{""},
			noPlan:           false,
			setupMocks:       func(deps *Dependencies) {},
			wantErr:          true,
			expectedErrorMsg: "task description cannot be empty",
		},
		{
			name:             "no arguments provided",
			args:             []string{},
			noPlan:           false,
			setupMocks:       func(deps *Dependencies) {},
			wantErr:          true,
			expectedErrorMsg: "task description is required",
		},
		{
			name:   "config loading fails",
			args:   []string{"Some task"},
			noPlan: false,
			setupMocks: func(deps *Dependencies) {
				deps.ConfigLoader.(*MockConfigLoader).On("Load").Return((*config.Config)(nil), errors.New("config error"))
			},
			wantErr:          true,
			expectedErrorMsg: "failed to load config",
		},
		{
			name:   "workflow execution fails",
			args:   []string{"Some task"},
			noPlan: false,
			setupMocks: func(deps *Dependencies) {
				cfg := &config.Config{WorkDir: "/tmp"}
				deps.ConfigLoader.(*MockConfigLoader).On("Load").Return(cfg, nil)
				deps.WorkflowEngine.(*MockWorkflowEngine).On("Run", mock.Anything, "Some task", true).Return(errors.New("workflow failed"))
			},
			wantErr:          true,
			expectedErrorMsg: "workflow failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock dependencies
			deps := &Dependencies{
				ConfigLoader:   &MockConfigLoader{},
				WorkflowEngine: &MockWorkflowEngine{},
				FileReader:     &MockFileReader{},
			}

			// Setup mocks
			tt.setupMocks(deps)

			// Test the workflow execution with dependency injection
			err := runWorkflowWithDependencies(context.Background(), tt.args, tt.noPlan, false, "", deps)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErrorMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify mock expectations
			deps.ConfigLoader.(*MockConfigLoader).AssertExpectations(t)
			deps.WorkflowEngine.(*MockWorkflowEngine).AssertExpectations(t)
			deps.FileReader.(*MockFileReader).AssertExpectations(t)
		})
	}
}

// TestRunWorkflowWithFileInput tests file input scenarios
func TestRunWorkflowWithFileInput(t *testing.T) {
	tests := []struct {
		name             string
		filename         string
		fileContent      []byte
		fileError        error
		noPlan           bool
		setupMocks       func(*Dependencies)
		wantErr          bool
		expectedErrorMsg string
	}{
		{
			name:        "successful file input",
			filename:    "task.md",
			fileContent: []byte("Implement payment gateway integration"),
			fileError:   nil,
			noPlan:      false,
			setupMocks: func(deps *Dependencies) {
				cfg := &config.Config{WorkDir: "/tmp"}
				deps.ConfigLoader.(*MockConfigLoader).On("Load").Return(cfg, nil)
				deps.FileReader.(*MockFileReader).On("ReadFile", "task.md").Return([]byte("Implement payment gateway integration"), nil)
				deps.WorkflowEngine.(*MockWorkflowEngine).On("Run", mock.Anything, "Implement payment gateway integration", true).Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "successful file input with --no-plan",
			filename:    "urgent.md",
			fileContent: []byte("Fix production bug immediately"),
			fileError:   nil,
			noPlan:      true,
			setupMocks: func(deps *Dependencies) {
				cfg := &config.Config{WorkDir: "/tmp"}
				deps.ConfigLoader.(*MockConfigLoader).On("Load").Return(cfg, nil)
				deps.FileReader.(*MockFileReader).On("ReadFile", "urgent.md").Return([]byte("Fix production bug immediately"), nil)
				deps.WorkflowEngine.(*MockWorkflowEngine).On("Run", mock.Anything, "Fix production bug immediately", false).Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "file not found",
			filename:    "missing.md",
			fileContent: nil,
			fileError:   os.ErrNotExist,
			setupMocks: func(deps *Dependencies) {
				deps.FileReader.(*MockFileReader).On("ReadFile", "missing.md").Return([]byte(nil), os.ErrNotExist)
			},
			wantErr:          true,
			expectedErrorMsg: "failed to read task file",
		},
		{
			name:        "empty file content",
			filename:    "empty.md",
			fileContent: []byte(""),
			fileError:   nil,
			setupMocks: func(deps *Dependencies) {
				deps.FileReader.(*MockFileReader).On("ReadFile", "empty.md").Return([]byte(""), nil)
			},
			wantErr:          true,
			expectedErrorMsg: "task description cannot be empty",
		},
		{
			name:        "whitespace-only file content",
			filename:    "whitespace.md",
			fileContent: []byte("   \n\t  \n"),
			fileError:   nil,
			setupMocks: func(deps *Dependencies) {
				deps.FileReader.(*MockFileReader).On("ReadFile", "whitespace.md").Return([]byte("   \n\t  \n"), nil)
			},
			wantErr:          true,
			expectedErrorMsg: "task description cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock dependencies
			deps := &Dependencies{
				ConfigLoader:   &MockConfigLoader{},
				WorkflowEngine: &MockWorkflowEngine{},
				FileReader:     &MockFileReader{},
			}

			// Setup mocks
			tt.setupMocks(deps)

			// This test will fail until we refactor runWorkflow to accept dependencies
			err := runWorkflowWithDependencies(context.Background(), []string{}, tt.noPlan, false, tt.filename, deps)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErrorMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify mock expectations
			deps.ConfigLoader.(*MockConfigLoader).AssertExpectations(t)
			deps.WorkflowEngine.(*MockWorkflowEngine).AssertExpectations(t)
			deps.FileReader.(*MockFileReader).AssertExpectations(t)
		})
	}
}

// TestSignalHandling tests interrupt signal behavior
func TestSignalHandling(t *testing.T) {
	t.Run("graceful shutdown on interrupt", func(t *testing.T) {
		// Create mock dependencies
		deps := &Dependencies{
			ConfigLoader:   &MockConfigLoader{},
			WorkflowEngine: &MockWorkflowEngine{},
			FileReader:     &MockFileReader{},
		}

		cfg := &config.Config{WorkDir: "/tmp"}
		deps.ConfigLoader.(*MockConfigLoader).On("Load").Return(cfg, nil)

		// Mock a long-running workflow that gets cancelled
		deps.WorkflowEngine.(*MockWorkflowEngine).On("Run", mock.Anything, "Long task", true).Return(context.Canceled)

		// Start the workflow in a goroutine
		errChan := make(chan error, 1)
		go func() {
			err := runWorkflowWithDependencies(context.Background(), []string{"Long task"}, false, false, "", deps)
			errChan <- err
		}()

		// Simulate interrupt signal after a short delay
		time.Sleep(10 * time.Millisecond)
		// This test structure shows what we want to test, but requires signal handling refactor
		// For now, we'll verify the context cancellation behavior

		select {
		case err := <-errChan:
			// Should receive context.Canceled error
			assert.Equal(t, context.Canceled, err)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Test timed out")
		}

		deps.ConfigLoader.(*MockConfigLoader).AssertExpectations(t)
		deps.WorkflowEngine.(*MockWorkflowEngine).AssertExpectations(t)
	})
}

// The runWorkflowWithDependencies function is now in workflow.go for reuse
