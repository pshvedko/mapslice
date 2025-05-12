package mapslice

import (
	"github.com/puzpuzpuz/xsync/v3"
)

type Subscription[K comparable, V any] struct {
	holder chan map[K][]V
	signal chan struct{}
	offset map[K]int
}

func (s Subscription[K, V]) Set(key K, values []V) {
	if len(values) == 0 {
		return
	}
	m := <-s.holder
	m[key] = values
	s.holder <- m
	select {
	case s.signal <- struct{}{}:
	default:
	}
}

func (s Subscription[K, V]) Load() (keys []K, values [][]V) {
	m := <-s.holder
	s.holder <- make(map[K][]V, len(s.offset))
	for key, slice := range m {
		keys, values, s.offset[key] = append(keys, key), append(values, slice[s.offset[key]:]), len(slice)
	}
	return
}

func (s Subscription[K, V]) Ready() <-chan struct{} {
	return s.signal
}

type Slice[K comparable, V any] struct {
	values        []V
	subscriptions []Subscription[K, V]
}

type MapSlice[K comparable, V any] struct {
	storage *xsync.MapOf[K, Slice[K, V]]
}

func NewMapSlice[K comparable, V any]() *MapSlice[K, V] {
	return &MapSlice[K, V]{
		storage: xsync.NewMapOf[K, Slice[K, V]](),
	}
}

func (m *MapSlice[K, V]) Append(key K, values ...V) {
	m.storage.Compute(key, func(slice Slice[K, V], loaded bool) (Slice[K, V], bool) {
		slice.values = append(slice.values, values...)
		for _, sub := range slice.subscriptions {
			sub.Set(key, slice.values)
		}
		return slice, false
	})
}

func (m *MapSlice[K, V]) Subscribe(keys ...K) Subscription[K, V] {
	sub := Subscription[K, V]{
		holder: make(chan map[K][]V, 1),
		signal: make(chan struct{}, 1),
		offset: make(map[K]int, len(keys)),
	}

	sub.holder <- make(map[K][]V, len(keys))

	for _, key := range keys {
		_, ok := sub.offset[key]
		if ok {
			continue
		}
		sub.offset[key] = 0
		m.storage.Compute(key, func(slice Slice[K, V], loaded bool) (Slice[K, V], bool) {
			if len(slice.values) > 0 {
				sub.Set(key, slice.values)
			}
			slice.subscriptions = append(slice.subscriptions, sub)
			return slice, false
		})
	}

	return sub
}

func (m *MapSlice[K, V]) Unsubscribe(sub Subscription[K, V]) {
	for key := range sub.offset {
		m.storage.Compute(key, func(slice Slice[K, V], loaded bool) (Slice[K, V], bool) {
			for i, x := range slice.subscriptions {
				if x.signal == sub.signal {
					n := len(slice.subscriptions)
					n--
					slice.subscriptions[i] = slice.subscriptions[n]
					slice.subscriptions = slice.subscriptions[:n]
					break
				}
			}
			return slice, len(slice.subscriptions) == 0 && len(slice.values) == 0
		})
	}
}

func (m *MapSlice[K, V]) Delete(keys ...K) {
	for _, key := range keys {
		m.storage.Delete(key)
	}
}

func (m *MapSlice[K, V]) Range(f func(key K, value []V) bool) {
	m.storage.Range(func(key K, slice Slice[K, V]) bool {
		return f(key, slice.values)
	})
}
