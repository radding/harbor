package mathparser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpression(t *testing.T) {
	assert := assert.New(t)

	res, err := trueExpr.Evaluate(nil)
	assert.NoError(err)
	assert.True(res)

	res, err = falseExpr.Evaluate(nil)
	assert.NoError(err)
	assert.False(res)
}

func TestExpressionWithOpTerm(t *testing.T) {
	assert := assert.New(t)

	opTerm := &OpTerm{
		RightExpr:       trueExpr,
		LogicalOperator: opPtr(OpAnd),
	}

	newExpression := createExpression(true)
	newExpression.OpTerm = opTerm

	res, err := newExpression.Evaluate(nil)
	assert.NoError(err)
	assert.True(res)
}
