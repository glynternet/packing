package parse_test

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/glynternet/packing/pkg/parse"
)

func TestNewPrefixedParser(t *testing.T) {
	testPrefix := "TEST_PREFIX"
	parseFn := parse.NewPrefixedParser(testPrefix)

	t.Run("errors", func(t *testing.T) {
		for _, test := range []struct {
			name string
			input string
		}{
			{
				name: "empty input",
			},
			{
				name: "non-matching prefix",
				input:"hey wassup",
			},
		}{
			t.Run(test.name, func(t *testing.T) {
				listName, err := parseFn(test.input)
				assert.Error(t, err)
				assert.Equal(t, "", listName)
			})
		}
	})

	t.Run("successes", func(t *testing.T) {
		for _, test := range []struct {
			name string
			input string
			listName string
		}{
			{
				name:"input equals prefix",
				input:testPrefix,
			},
			{
				name:"prefix exists with valid suffix",
				input:testPrefix + "crazy suffix",
				listName: "crazy suffix",
			},
		}{
			t.Run(test.name, func(t *testing.T) {
				listName, err := parseFn(test.input)
				assert.NoError(t, err)
				assert.Equal(t, test.listName, listName)
			})
		}
	})
}
