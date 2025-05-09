package mapslice_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pshvedko/mapslice"
)

func TestMapSlice_Append(t *testing.T) {
	m := mapslice.NewMapSlice[int, string]()

	s123 := m.Subscribe(1, 2, 3)

	m.Append(1, "1a")
	m.Append(1, "1b")
	m.Append(1, "1c")

	select {
	case <-s123.Ready():
		require.True(t, true)
	default:
		require.True(t, false)
	}

	k, v := s123.Load()

	require.ElementsMatch(t, []int{1}, k)
	require.ElementsMatch(t, [][]string{{"1a", "1b", "1c"}}, v)

	select {
	case <-s123.Ready():
		require.True(t, false)
	default:
		require.True(t, true)
	}

	k, v = s123.Load()

	require.ElementsMatch(t, []int{}, k)
	require.ElementsMatch(t, [][]string{}, v)

	m.Append(1, "1d")
	m.Append(2, "2a")
	m.Append(2, "2b")
	m.Append(2, "2c")
	m.Append(3, "3a")
	m.Append(4, "4a")
	m.Append(5, "5a")
	m.Append(5, "5b")

	m.Delete(2, 3)

	select {
	case <-s123.Ready():
		require.True(t, true)
	default:
		require.True(t, false)
	}

	k, v = s123.Load()

	require.ElementsMatch(t, []int{1, 2, 3}, k)
	require.ElementsMatch(t, [][]string{{"1d"}, {"2a", "2b", "2c"}, {"3a"}}, v)

	s145 := m.Subscribe(1, 4, 5)

	m.Delete(1)

	select {
	case <-s145.Ready():
		require.True(t, true)
	default:
		require.True(t, false)
	}

	k, v = s145.Load()

	require.ElementsMatch(t, []int{1, 4, 5}, k)
	require.ElementsMatch(t, [][]string{{"1a", "1b", "1c", "1d"}, {"4a"}, {"5a", "5b"}}, v)

	m.Append(1, "1a")
	m.Append(2, "2a")
	m.Append(3, "3a")
	m.Append(4, "4b", "4c")
	m.Append(5, "5c", "5d")

	m.Delete(1, 2, 3)

	select {
	case <-s123.Ready():
		require.True(t, false)
	default:
		require.True(t, true)
	}

	select {
	case <-s145.Ready():
		require.True(t, true)
	default:
		require.True(t, false)
	}

	k, v = s145.Load()

	require.ElementsMatch(t, []int{4, 5}, k)
	require.ElementsMatch(t, [][]string{{"4b", "4c"}, {"5c", "5d"}}, v)

	s345 := m.Subscribe(3, 4, 5)

	m.Unsubscribe(s123)
	m.Unsubscribe(s145)
	m.Unsubscribe(s345)

	m.Delete(5, 4)
}
