package mathparser

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComparison(t *testing.T) {
	for _, cse := range tests {
		for op, value := range cse.results {
			t.Run(fmt.Sprintf("%s %s %s", cse.valueA, op, cse.valueB), func(t *testing.T) {
				assert := assert.New(t)
				comp := &ComparisonExpression{
					Left:               cse.valueA,
					ComparisonOperator: &op,
					Right:              cse.valueB,
				}
				res, err := comp.Evaluate(nil)
				assert.NoError(err)
				assert.Equal(value, res)
			})

		}
	}
}
