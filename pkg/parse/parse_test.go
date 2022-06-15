package parse_test

import (
	"testing"

	"github.com/glynternet/packing/pkg/parse"
	"github.com/stretchr/testify/assert"
)

func TestNewPrefixedParser(t *testing.T) {
	testPrefix := "TEST_PREFIX"
	parseFn := parse.NewPrefixedParser(testPrefix)

	t.Run("non-matching", func(t *testing.T) {
		for _, test := range []struct {
			name  string
			input string
		}{
			{
				name: "empty input",
			},
			{
				name:  "non-matching prefix",
				input: "hey wassup",
			},
		} {
			t.Run(test.name, func(t *testing.T) {
				_, match := parseFn(test.input)
				assert.False(t, match)
			})
		}
	})

	t.Run("matching", func(t *testing.T) {
		for _, test := range []struct {
			name   string
			input  string
			suffix string
		}{
			{
				name:  "input equals prefix",
				input: testPrefix,
			},
			{
				name:   "prefix exists with valid suffix",
				input:  testPrefix + "crazy suffix",
				suffix: "crazy suffix",
			},
		} {
			t.Run(test.name, func(t *testing.T) {
				listName, match := parseFn(test.input)
				assert.True(t, match)
				assert.Equal(t, test.suffix, listName)
			})
		}
	})
}
