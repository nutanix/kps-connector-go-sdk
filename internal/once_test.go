package internal

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOnce(t *testing.T) {
	t.Run("TryDo gets called once if instantiation succeeds", func(t *testing.T) {
		calls := 0
		expectedCalls := 1
		var c Once
		for i := 1; i < 5; i++ {
			err := c.TryDo(func() error {
				calls++
				return nil
			})
			assert.NoError(t, err)
		}
		assert.Equal(t, expectedCalls, calls)
	})

	t.Run("TryDo gets called n times if instantiations fail n times", func(t *testing.T) {
		calls := 0
		expectedCalls := 5
		var c Once
		for i := 0; i < 5; i++ {
			err := c.TryDo(func() error {
				calls++
				return fmt.Errorf("error")
			})
			assert.Error(t, err)
			assert.EqualError(t, err, "error")
		}
		assert.Equal(t, expectedCalls, calls)
	})

	t.Run("TryDo does not get called after instantiation succeeds", func(t *testing.T) {
		calls := 0
		expectedCalls := 6
		var c Once
		for i := 0; i < 5; i++ {
			err := c.TryDo(func() error {
				calls++
				return fmt.Errorf("error")
			})
			assert.Error(t, err)
			assert.EqualError(t, err, "error")
		}

		for i := 1; i < 5; i++ {
			err := c.TryDo(func() error {
				calls++
				return nil
			})
			assert.NoError(t, err)
		}
		assert.Equal(t, expectedCalls, calls)
	})
}
