package runners

import (
	"fmt"
	"sync"
	"testing"
	"time"

	plugins "github.com/radding/harbor-plugins"
	"github.com/radding/harbor-plugins/proto"
	"github.com/radding/harbor/internal/workspaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockPlugin struct {
	mock.Mock
	errorOut bool
	mockTask *mockTask
}

type mockTask struct {
	mock.Mock
	state    plugins.TaskStatus
	exitCode int64
}

func (m *mockTask) transition(state plugins.TaskStatus) {
	m.state = state
}

func (m *mockTask) Status() plugins.RunResponse {
	m.Called()
	return plugins.RunResponse{
		Status:   proto.RunStatus(m.state),
		ExitCode: m.exitCode,
	}
}

func (m *mockTask) Stop(signal int64, timeoutMS int64) error {
	m.Called(signal, timeoutMS)
	return nil
}

func (m *mockTask) Wait() plugins.RunResponse {
	m.Called()
	var resp plugins.RunResponse
	resp.Status = proto.RunStatus_FINISHED
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
	}
	// for resp = m.Status(); resp.Status != proto.RunStatus_CANCELED && resp.Status != proto.RunStatus_FINISHED && resp.Status != proto.RunStatus_CRASHED; {

	// }
	return resp
}

func (m *MockPlugin) Run(req plugins.RunRequest) (plugins.ClientTask, error) {
	m.Called(req)
	if m.errorOut && req.CommandName == "test4" {
		return m.mockTask, fmt.Errorf("some error happened")
	}
	return m.mockTask, nil
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
	mockT := &mockTask{}
	mockT.On("Wait", mock.Anything)
	mockT.On("Status", mock.Anything)
	mockT.On("Stop", mock.Anything, mock.Anything)
	mockPlugin := &MockPlugin{
		mockTask: mockT,
	}
	called := false
	calledWith := ""

	rCtx := newRunContext()
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
	recipe.Run([]string{}, mockFetcher, rCtx)
	go func() {
		time.Sleep(5 * time.Second)
		mockT.transition(plugins.FINISHED)
	}()
	go func() {
		time.Sleep(20 * time.Second)
		rCtx.cancelFunc()
	}()
	assert.True(called)
	assert.Equal(calledWith, "testRunner")
	mockPlugin.AssertCalled(t, "Run", mock.Anything)
}

func TestRunsInCorrectOrder(t *testing.T) {
	assert := assert.New(t)
	mockT := &mockTask{}
	mockT.On("Wait", mock.Anything)
	mockT.On("Status", mock.Anything)
	mockT.On("Stop", mock.Anything, mock.Anything)
	mockPlugin := &MockPlugin{
		mockTask: mockT,
	}
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

	rCtx := newRunContext()
	err := recipe.Run([]string{}, mockFetcher, rCtx)
	assert.NoError(err)
	mockPlugin.AssertNumberOfCalls(t, "Run", 4)

	arg1 := mockPlugin.Calls[0].Arguments[0].(plugins.RunRequest)
	arg2 := mockPlugin.Calls[1].Arguments[0].(plugins.RunRequest)
	arg3 := mockPlugin.Calls[2].Arguments[0].(plugins.RunRequest)
	arg4 := mockPlugin.Calls[3].Arguments[0].(plugins.RunRequest)

	assertFirstOneIscorrect := arg1.CommandName == "test4" || arg1.CommandName == "test3"
	assertSecondOneIscorrect := arg2.CommandName == "test4" || arg2.CommandName == "test3"
	assert.True(assertFirstOneIscorrect)
	assert.True(assertSecondOneIscorrect)
	assert.Equal("test2", arg3.CommandName)
	assert.Equal("test1", arg4.CommandName)
}

func TestEachStepOnlyRunsOnce(t *testing.T) {
	assert := assert.New(t)
	mockT := &mockTask{}
	mockT.On("Wait", mock.Anything)
	mockT.On("Status", mock.Anything)
	mockT.On("Stop", mock.Anything, mock.Anything)
	mockPlugin := &MockPlugin{
		mockTask: mockT,
	}
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

	rCtx := newRunContext()
	err := recipe.Run([]string{}, mockFetcher, rCtx)
	assert.NoError(err)
	mockPlugin.AssertNumberOfCalls(t, "Run", 4)

	arg1 := mockPlugin.Calls[0].Arguments[0].(plugins.RunRequest)
	arg2 := mockPlugin.Calls[1].Arguments[0].(plugins.RunRequest)
	arg3 := mockPlugin.Calls[2].Arguments[0].(plugins.RunRequest)
	arg4 := mockPlugin.Calls[3].Arguments[0].(plugins.RunRequest)

	assertFirstOneIscorrect := arg2.CommandName == "test3" || arg2.CommandName == "test2"
	assertSecondOneIscorrect := arg3.CommandName == "test3" || arg3.CommandName == "test2"
	assert.Equal("test4", arg1.CommandName)
	assert.True(assertFirstOneIscorrect)
	assert.True(assertSecondOneIscorrect)
	assert.Equal("test1", arg4.CommandName)
}

func TestWillReturnErrorsFromChildren(t *testing.T) {
	assert := assert.New(t)
	mockT := &mockTask{}
	mockT.On("Wait", mock.Anything)
	mockT.On("Status", mock.Anything)
	mockT.On("Stop", mock.Anything, mock.Anything)
	mockPlugin := &MockPlugin{errorOut: true, mockTask: mockT}
	mockFetcher := func(name string) (plugins.PluginClient, error) {
		return mockPlugin, nil
	}
	mockPlugin.On("Run", mock.Anything).Return(nil)
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

	rCtx := newRunContext()
	err := recipe.Run([]string{}, mockFetcher, rCtx)
	assert.Error(err)
	// call test4 and test1
	mockPlugin.AssertNumberOfCalls(t, "Run", 2)
}
