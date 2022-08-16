package utils

import (
	"reflect"
)

type RoundRobin struct {
	data []interface{}
	next int
}

func (r *RoundRobin) Next() interface{} {
	v := r.data[r.next]
	r.next = (r.next + 1) % r.Len()
	return v
}

func (r *RoundRobin) Append(v interface{}) {
	r.data = append(r.data, v)
}

func (r *RoundRobin) Remove(v interface{}) bool {
	found := -1
	for index, item := range r.data {
		if reflect.DeepEqual(item, v) {
			found = index
			break
		}
	}
	if found == -1 {
		return false
	}

	r.data = append(r.data[:found], r.data[found+1:]...)
	if found < r.next {
		r.next--
	}
	if r.next == r.Len() {
		r.next = 0
	}
	return true
}

func (r *RoundRobin) Len() int {
	return len(r.data)
}

func NewRoundRobin() *RoundRobin {
	return &RoundRobin{make([]interface{}, 0), 0}
}
