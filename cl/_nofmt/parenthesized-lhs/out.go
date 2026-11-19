package main

type cell struct {
	value int
}

func assign(p *int, values []int, array *[1]int, entries map[int]int, c *cell, ch chan int) {
	*p = 1
	values[0] += 2
	array[0]++
	entries[0], c.value = *p, values[0]
	for c.value, _ = range values {
	}
	select {
	case entries[0] = <-ch:
	default:
	}
}
