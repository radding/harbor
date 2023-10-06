package mathparser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanParseFine(t *testing.T) {
	assert := assert.New(t)
	expr, err := Parse("$THING1 >= \"Test\"")
	v := &ValueLookup{
		returns: &Value{
			StringValue: stringPtr("\"Test\""),
		},
	}
	assert.NoError(err)
	assert.Equal("THING1", *expr.Left.Left.EnvVar)
	assert.Equal(OpGte, *expr.Left.ComparisonOperator)
	assert.Equal("\"Test\"", *expr.Left.Right.StringValue)

	res, err := expr.Evaluate(v)
	assert.NoError(err)
	assert.True(res)
}

func TestCanParseComplex(t *testing.T) {
	assert := assert.New(t)
	expr, err := Parse("$THING1 >= \"Test\" && 0 == 0 || 3 == 10")

	assert.NoError(err)
	v := &ValueLookup{
		returns: &Value{
			StringValue: stringPtr("\"Test\""),
		},
	}
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
