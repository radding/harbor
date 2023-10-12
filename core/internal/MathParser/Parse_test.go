package mathparser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanParseFine(t *testing.T) {
	assert := assert.New(t)
	expr, err := Parse("${{env.OS}} >= \"Test\"")
	v := &ValueLookup{
		returns: &Value{
			StringValue: stringPtr("\"Test\""),
		},
	}
	v.On("GetValue", "env", "OS").Once()
	assert.NoError(err)
	assert.Equal("env", *expr.Left.Left.Variable.Provider)
	assert.Equal("OS", *expr.Left.Left.Variable.Value)
	assert.Equal(OpGte, *expr.Left.ComparisonOperator)
	assert.Equal("\"Test\"", *expr.Left.Right.StringValue)

	res, err := expr.Evaluate(v)
	assert.NoError(err)
	assert.True(res)
}

func TestCanParseComplex(t *testing.T) {
	assert := assert.New(t)
	expr, err := Parse("${{env.OS}} >= \"Test\" && 0 == 0 || 3 == 10")

	assert.NoError(err)
	v := &ValueLookup{
		returns: &Value{
			StringValue: stringPtr("\"Test\""),
		},
	}
	v.On("GetValue", "env", "OS").Once()
	res, err := expr.Evaluate(v)
	assert.NoError(err)
	assert.True(res)
}

func floatPtr(val float64) *float64 {
	return &val
}

func stringPtr(val string) *string {
	return &val
}

func boolPtr(val bool) *bool {
	return &val
}
