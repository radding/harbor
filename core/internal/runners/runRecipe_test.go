package runners

import (
	"context"
	"fmt"
	"sync"
	"testing"

	plugins "github.com/radding/harbor-plugins"
	"github.com/radding/harbor/internal/workspaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPlugin struct {
	mock.Mock
	errorOut bool
}

func (m *MockPlugin) Run(ctx context.Context, req plugins.RunRequest) (plugins.RunResponse, error) {
	m.Called(req)
	if m.errorOut && req.CommandName == "test4" {
		return plugins.RunResponse{}, fmt.Errorf("some error happened")
	}
	return plugins.RunResponse{}, nil
}

func (m *MockPlugin) Install() (*plugins.PluginDefinition, error) {
	m.Called()
	return nil, nil
}

func (m *MockPlugin) Kill() {
	m.Called()
}

func TestCanRunTestFine(t *testing.T) {
	assert := assert.New(t)
	mockPlugin := &MockPlugin{}
	called := false
	calledWith := ""
	mockFetcher := func(name string) (plugins.PluginClient, error) {
		called = true
		calledWith = name
		return mockPlugin, nil
	}
	mockPlugin.On("Run", mock.Anything)
	recipe := &RunRecipe{
		CommandName: "test",
		wg:          &sync.WaitGroup{},
		done:        false,
		runConfig: &workspaces.Command{
			Type:    "testRunner",
			Command: "some command",
		},
		pkgObject: workspaces.WorkspaceConfig{},
	}
	recipe.Run([]string{}, mockFetcher)
	assert.True(called)
	assert.Equal(calledWith, "testRunner")
	mockPlugin.AssertCalled(t, "Run", mock.Anything)
}

func TestRunsInCorrectOrder(t *testing.T) {
	assert := assert.New(t)
	mockPlugin := &MockPlugin{}
	mockFetcher := func(name string) (plugins.PluginClient, error) {
		return mockPlugin, nil
	}
	mockPlugin.On("Run", mock.Anything)
	recipe := &RunRecipe{
		CommandName: "test1",
		wg:          &sync.WaitGroup{},
		done:        false,
		runConfig: &workspaces.Command{
			Type:    "testRunner",
			Command: "some command",
		},
		pkgObject: workspaces.WorkspaceConfig{},
		Needs: []*RunRecipe{
			{
				CommandName: "test2",
				wg:          &sync.WaitGroup{},
				done:        false,
				runConfig: &workspaces.Command{
					Type:    "testRunner",
					Command: "some command",
				},
				Needs: []*RunRecipe{
					{
						CommandName: "test4",
						wg:          &sync.WaitGroup{},
						done:        false,
						runConfig: &workspaces.Command{
							Type:    "testRunner",
							Command: "some command",
						},
					},
				},
			},
			{
				CommandName: "test3",
				wg:          &sync.WaitGroup{},
				done:        false,
				runConfig: &workspaces.Command{
					Type:    "testRunner",
					Command: "some command",
				},
			},
		},
	}

	err := recipe.Run([]string{}, mockFetcher)
	assert.NoError(err)
	mockPlugin.AssertNumberOfCalls(t, "Run", 4)

	arg1 := mockPlugin.Calls[0].Arguments[1].(plugins.RunRequest)
	arg2 := mockPlugin.Calls[1].Arguments[1].(plugins.RunRequest)
	arg3 := mockPlugin.Calls[2].Arguments[1].(plugins.RunRequest)
	arg4 := mockPlugin.Calls[3].Arguments[1].(plugins.RunRequest)

	assertFirstOneIscorrect := arg1.CommandName == "test4" || arg1.CommandName == "test3"
	assertSecondOneIscorrect := arg2.CommandName == "test4" || arg2.CommandName == "test3"
	assert.True(assertFirstOneIscorrect)
	assert.True(assertSecondOneIscorrect)
	assert.Equal("test2", arg3.CommandName)
	assert.Equal("test1", arg4.CommandName)
}

func TestEachStepOnlyRunsOnce(t *testing.T) {
	assert := assert.New(t)
	mockPlugin := &MockPlugin{}
	mockFetcher := func(name string) (plugins.PluginClient, error) {
		return mockPlugin, nil
	}
	mockPlugin.On("Run", mock.Anything)
	step4 := &RunRecipe{
		CommandName: "test4",
		wg:          &sync.WaitGroup{},
		done:        false,
		runConfig: &workspaces.Command{
			Type:    "testRunner",
			Command: "some command",
		},
	}
	recipe := &RunRecipe{
		CommandName: "test1",
		wg:          &sync.WaitGroup{},
		done:        false,
		runConfig: &workspaces.Command{
			Type:    "testRunner",
			Command: "some command",
		},
		pkgObject: workspaces.WorkspaceConfig{},
		Needs: []*RunRecipe{
			{
				CommandName: "test2",
				wg:          &sync.WaitGroup{},
				done:        false,
				runConfig: &workspaces.Command{
					Type:    "testRunner",
					Command: "some command",
				},
				Needs: []*RunRecipe{
					step4,
				},
			},
			{
				CommandName: "test3",
				wg:          &sync.WaitGroup{},
				done:        false,
				runConfig: &workspaces.Command{
					Type:    "testRunner",
					Command: "some command",
				},
				Needs: []*RunRecipe{
					step4,
				},
			},
		},
	}

	err := recipe.Run([]string{}, mockFetcher)
	assert.NoError(err)
	mockPlugin.AssertNumberOfCalls(t, "Run", 4)

	arg1 := mockPlugin.Calls[0].Arguments[1].(plugins.RunRequest)
	arg2 := mockPlugin.Calls[1].Arguments[1].(plugins.RunRequest)
	arg3 := mockPlugin.Calls[2].Arguments[1].(plugins.RunRequest)
	arg4 := mockPlugin.Calls[3].Arguments[1].(plugins.RunRequest)

	assertFirstOneIscorrect := arg2.CommandName == "test3" || arg2.CommandName == "test2"
	assertSecondOneIscorrect := arg3.CommandName == "test3" || arg3.CommandName == "test2"
	assert.Equal("test4", arg1.CommandName)
	assert.True(assertFirstOneIscorrect)
	assert.True(assertSecondOneIscorrect)
	assert.Equal("test1", arg4.CommandName)
}

// func TestWillReturnErrorsFromChildren(t *testing.T) {
// assert := assert.New(t)
// mockPlugin := &MockPlugin{errorOut: true}
// mockFetcher := func(name string) (plugins.PluginClient, error) {
// return mockPlugin, nil
// }
// mockPlugin.On("Run", mock.Anything).Return(nil)
// recipe := &RunRecipe{
// CommandName: "test1",
// wg:          &sync.WaitGroup{},
// done:        false,
// runConfig: &workspaces.Command{
// Type:    "testRunner",
// Command: "some command",
// },
// pkgObject: workspaces.WorkspaceConfig{},
// Needs: []*RunRecipe{
// {
// CommandName: "test2",
// wg:          &sync.WaitGroup{},
// done:        false,
// runConfig: &workspaces.Command{
// Type:    "testRunner",
// Command: "some command",
// },
// Needs: []*RunRecipe{
// {
// CommandName: "test4",
// wg:          &sync.WaitGroup{},
// done:        false,
// runConfig: &workspaces.Command{
// Type:    "testRunner",
// Command: "some command",
// },
// },
// },
// },
// {
// CommandName: "test3",
// wg:          &sync.WaitGroup{},
// done:        false,
// runConfig: &workspaces.Command{
// Type:    "testRunner",
// Command: "some command",
// },
// },
// },
// }

// err := recipe.Run([]string{}, mockFetcher)
// assert.Error(err)
// // call test4 and test1
// mockPlugin.AssertNumberOfCalls(t, "Run", 2)
// }
