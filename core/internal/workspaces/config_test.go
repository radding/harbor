package workspaces

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

var testStr = `
Cond:
  - $TEST_ENV == "hello"
  - $TEST_ENV2 != "help"
  - $TEST_ENV3 > 0
  - $TEST_ENV4 >= 0
  - $TEST_ENV5 < 0
  - $TEST_ENV6 <= 0
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
