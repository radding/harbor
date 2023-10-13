package workspaces

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

var testStr = `
Cond:
  - ${{env.TEST_ENV}} == "hello"
  - ${{env.TEST_ENV2}} != "help"
  - ${{env.TEST_ENV3}} > 0
  - ${{env.TEST_ENV4}} >= 0
  - ${{env.TEST_ENV5}} < 0
  - ${{env.TEST_ENV6}} <= 0
`

func TestCanParseRunCondition(t *testing.T) {
	assert := assert.New(t)

	testVal := &struct {
		Cond []*RunCondition `yaml:"Cond"`
	}{}

	err := yaml.Unmarshal([]byte(testStr), &testVal)
	assert.NoError(err)

	assert.Len(testVal.Cond, 6)

}
