package mapslice

import (
	"github.com/puzpuzpuz/xsync/v3"
)

type Subscription[K comparable, V any] struct {
	locker chan struct{}
	signal chan struct{}
	update map[K][]V
	offset map[K]int
}

func (s Subscription[K, V]) Set(key K, values []V) {
	if len(values) > 0 {
		s.locker <- struct{}{}
		i, ok := s.offset[key]
		if ok {
			s.update[key] = values[i:]
			select {
			case s.signal <- struct{}{}:
			default:
			}
		}
		<-s.locker
	}
}

func (s Subscription[K, V]) Load() (keys []K, values [][]V) {
	s.locker <- struct{}{}
	for key, slice := range s.update {
		keys, values = append(keys, key), append(values, slice)
		s.offset[key] += len(slice)
	}
	clear(s.update)
	<-s.locker
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
	m.storage.Compute(key, func(a Slice[K, V], loaded bool) (Slice[K, V], bool) {
		a.values = append(a.values, values...)
		for _, s := range a.subscriptions {
			s.Set(key, a.values)
		}
		return a, false
	})
}

func (m *MapSlice[K, V]) Subscribe(keys ...K) Subscription[K, V] {
	s := Subscription[K, V]{
		locker: make(chan struct{}, 1),
		signal: make(chan struct{}, 1),
		update: make(map[K][]V, len(keys)),
		offset: make(map[K]int, len(keys)),
	}

	for _, key := range keys {
		_, ok := s.offset[key]
		if ok {
			continue
		}
		s.offset[key] = 0
		m.storage.Compute(key, func(a Slice[K, V], loaded bool) (Slice[K, V], bool) {
			s.Set(key, a.values)
			a.subscriptions = append(a.subscriptions, s)
			return a, false
		})
	}

	return s
}

func (m *MapSlice[K, V]) Unsubscribe(s Subscription[K, V]) {
	for key := range s.offset {
		m.storage.Compute(key, func(a Slice[K, V], loaded bool) (Slice[K, V], bool) {
			for i, x := range a.subscriptions {
				if x.signal == s.signal {
					n := len(a.subscriptions)
					n--
					a.subscriptions[i] = a.subscriptions[n]
					a.subscriptions = a.subscriptions[:n]
					break
				}
			}
			return a, false
		})
	}

	close(s.locker)
	close(s.signal)
	clear(s.update)
	clear(s.offset)
}

func (m *MapSlice[K, V]) Delete(keys ...K) {
	for _, key := range keys {
		m.storage.Delete(key)
	}
}
