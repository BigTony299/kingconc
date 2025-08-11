package kingconc_test

import (
	"testing"

	"github.com/BigTony299/kingconc"
	"github.com/stretchr/testify/assert"
)

func TestSome(t *testing.T) {
	assert := assert.New(t)

	const v uint = 123

	opt := kingconc.Some(v)

	assert.True(opt.Some(), "should be Some")
	assert.False(opt.None(), "should not be None")
	assert.Equal(v, opt.Raw(), "should contain `v`")
}

func TestNone(t *testing.T) {
	assert := assert.New(t)

	var zeroVal uint

	opt := kingconc.None[uint]()

	assert.False(opt.Some(), "should not be Some")
	assert.True(opt.None(), "should be None")
	assert.Equal(zeroVal, opt.Raw(), "should contain Zero value")
}

func TestOption_Get(t *testing.T) {
	assert := assert.New(t)

	t.Run("some", func(t *testing.T) {
		const v string = "abc"

		opt := kingconc.Some(v)

		actual, ok := opt.Get()

		assert.True(ok)
		assert.Equal(v, actual)
	})

	t.Run("none", func(t *testing.T) {
		var zeroVal string

		opt := kingconc.None[string]()

		actual, ok := opt.Get()

		assert.False(ok)
		assert.Equal(zeroVal, actual)
	})
}

func TestOption_GetOrDefault(t *testing.T) {
	assert := assert.New(t)

	t.Run("some", func(t *testing.T) {
		const v string = "abc"

		opt := kingconc.Some(v)

		actual := opt.GetOrDefault("def")

		assert.Equal(v, actual)
	})

	t.Run("none", func(t *testing.T) {
		const defVal string = "def"

		opt := kingconc.None[string]()

		actual := opt.GetOrDefault(defVal)

		assert.Equal(defVal, actual)
	})
}
