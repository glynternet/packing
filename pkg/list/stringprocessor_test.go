package list

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessorGroup_Process(t *testing.T) {
	t.Run("returns error when no processors configured", func(t *testing.T) {
		assert.EqualError(t, ProcessorGroup{}.Process(""), "no processors configured")
	})

	t.Run("no matches returns error", func(t *testing.T) {
		assert.EqualError(t, ProcessorGroup{func(s string) (bool, error) {
			return false, nil
		}}.Process("foo"), "no processor matched value: \"foo\"")
	})

	t.Run("returns immediately after first true from processor", func(t *testing.T) {
		assert.NoError(t, ProcessorGroup{func(string) (bool, error) {
			return true, nil
		}, func(string) (bool, error) {
			assert.FailNow(t, "should not be called")
			return false, nil
		}}.Process(""))
	})

	t.Run("passes string to all processors", func(t *testing.T) {
		var vals []string
		assert.NoError(t, ProcessorGroup{
			func(s string) (bool, error) {
				vals = append(vals, s)
				return false, nil
			}, func(s string) (bool, error) {
				vals = append(vals, s)
				return true, nil
			}}.Process("foo"))
		assert.Equal(t, []string{"foo", "foo"}, vals)
	})

	t.Run("returns first encountered error", func(t *testing.T) {
		expectedErr := errors.New("err")
		var firstProcessorCalled bool
		assert.Equal(t, expectedErr, ProcessorGroup{
			func(s string) (bool, error) {
				firstProcessorCalled = true
				return false, nil
			}, func(s string) (bool, error) {
				return false, expectedErr
			}}.Process("foo"))
		assert.True(t, firstProcessorCalled)
	})
}
