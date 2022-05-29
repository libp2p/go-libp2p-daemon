package utils

type RoundRobin struct {
	data []interface{}
	next int
}

func (r *RoundRobin) Next() interface{} {
	v := r.data[r.next]
	r.next = (r.next + 1) % r.Len()
	return v
}

func (r *RoundRobin) Push(v interface{}) {
	r.data = append(r.data, v)
}

func (r *RoundRobin) Len() int {
	return len(r.data)
}

func NewRoundRobin() *RoundRobin {
	return &RoundRobin{make([]interface{}, 0), 0}
}
