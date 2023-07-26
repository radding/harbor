package runners

import (
	"testing"

	"github.com/radding/harbor/internal/workspaces"
	"github.com/stretchr/testify/assert"
)

var defaultConf = workspaces.WorkspaceConfig{
	Name:     "Root",
	Commands: map[string]workspaces.Command{},
}

func init() {
	defaultConf.AddSubPackage("subPackageA", workspaces.WorkspaceConfig{
		Commands: map[string]workspaces.Command{
			"command1": {
				Type:    "test1",
				Command: "test",
				Dependencies: []workspaces.Dependency{
					{PackageName: ".", CommandName: "command2"},
				},
			},
			"command2": {
				Type:    "test1",
				Command: "test",
			},
		},
	})
	defaultConf.AddSubPackage("subPackageB", workspaces.WorkspaceConfig{
		Commands: map[string]workspaces.Command{
			"command1": {
				Type:    "test1",
				Command: "test",
				Dependencies: []workspaces.Dependency{
					{PackageName: "subPackageC", CommandName: "command3"},
				},
			},
		},
	})
	defaultConf.AddSubPackage("subPackageC", workspaces.WorkspaceConfig{
		Commands: map[string]workspaces.Command{
			"command3": {
				Type:         "test1",
				Command:      "test",
				Dependencies: []workspaces.Dependency{},
			},
		},
	})
}

func findInArray(arr []*RunRecipe, runRecipeToFind *RunRecipe) *RunRecipe {
	for _, rec := range arr {
		if rec.Pkg == runRecipeToFind.Pkg && rec.CommandName == runRecipeToFind.CommandName {
			return rec
		}
	}
	return nil
}

func (r *RunRecipe) assertMatches(other *RunRecipe) bool {
	commandNameMatches := r.CommandName == other.CommandName
	pkgMatches := r.Pkg == other.Pkg
	lenOfDepsMatch := len(r.Needs) == len(other.Needs)
	depsMatch := true
	for _, dep := range r.Needs {
		recToTest := findInArray(other.Needs, dep)
		if recToTest == nil {
			return false
		}
		depsMatch = depsMatch && recToTest.assertMatches(dep)
	}
	return commandNameMatches && pkgMatches && lenOfDepsMatch && depsMatch
}

func TestWillCreateASimpleRunRecipeFromSimpleConfig(t *testing.T) {
	assert := assert.New(t)
	conf := workspaces.WorkspaceConfig{
		Name:     "Root",
		Commands: map[string]workspaces.Command{},
	}
	conf.AddSubPackage("subPackageA", workspaces.WorkspaceConfig{
		Commands: map[string]workspaces.Command{
			"command1": {
				Type:    "test1",
				Command: "test",
				Dependencies: []workspaces.Dependency{
					{PackageName: ".", CommandName: "command2"},
				},
			},
			"command2": {
				Type:    "test1",
				Command: "test",
			},
		},
	})
	recipe, err := getRootRecipe("command1", conf)
	assert.NoError(err)
	assert.Equal(recipe.CommandName, "command1")
	expectedRecipe := &RunRecipe{
		CommandName: "command1",
		Pkg:         "Root",
		Needs: []*RunRecipe{
			{
				CommandName: "command1",
				Pkg:         "subPackageA",
				Needs: []*RunRecipe{
					{
						CommandName: "command2",
						Pkg:         "subPackageA",
						Needs:       []*RunRecipe{},
					},
				},
			},
		},
	}
	assert.True(recipe.assertMatches(expectedRecipe))
}

func TestMakeMoreComplexDeps(t *testing.T) {
	assert := assert.New(t)

	recipe, err := getRootRecipe("command1", defaultConf)
	assert.NoError(err)
	expectedRecipe := &RunRecipe{
		CommandName: "command1",
		Pkg:         "Root",
		Needs: []*RunRecipe{
			{
				CommandName: "command1",
				Pkg:         "subPackageA",
				Needs: []*RunRecipe{
					{
						CommandName: "command2",
						Pkg:         "subPackageA",
						Needs:       []*RunRecipe{},
					},
				},
			},
			{
				CommandName: "command1",
				Pkg:         "subPackageB",
				Needs: []*RunRecipe{
					{
						CommandName: "command3",
						Pkg:         "subPackageC",
						Needs:       []*RunRecipe{},
					},
				},
			},
		},
	}
	assert.True(recipe.assertMatches(expectedRecipe))
}

func TestThrowsErrorIfCommandIsNotFound(t *testing.T) {
	assert := assert.New(t)
	_, err := getRootRecipe("doesNotExsist", defaultConf)
	assert.Error(err)
}

func TestThrowsErrorOnCircularDependency(t *testing.T) {
	assert := assert.New(t)

	conf := workspaces.WorkspaceConfig{
		Name:     "Root",
		Commands: map[string]workspaces.Command{},
	}
	conf.AddSubPackage("subPackageA", workspaces.WorkspaceConfig{
		Commands: map[string]workspaces.Command{
			"command1": {
				Type:    "test1",
				Command: "test",
				Dependencies: []workspaces.Dependency{
					{PackageName: "subPackageB", CommandName: "command2"},
				},
			},
		},
	})
	conf.AddSubPackage("subPackagB", workspaces.WorkspaceConfig{
		Commands: map[string]workspaces.Command{
			"command2": {
				Type:    "test1",
				Command: "test",
				Dependencies: []workspaces.Dependency{
					{PackageName: "subPackageC", CommandName: "command3"},
				},
			},
		},
	})

	conf.AddSubPackage("subPackagC", workspaces.WorkspaceConfig{
		Commands: map[string]workspaces.Command{
			"command2": {
				Type:    "test1",
				Command: "test",
				Dependencies: []workspaces.Dependency{
					{PackageName: "subPackageA", CommandName: "command1"},
				},
			},
		},
	})

	_, err := getRootRecipe("command1", conf)
	assert.Error(err)
}
