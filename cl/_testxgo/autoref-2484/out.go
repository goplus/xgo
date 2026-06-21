package main

import "fmt"

type Person struct {
	Name string
	Age  int
}
type Point struct {
	X int
	Y int
}

func processPerson(p *Person) {
	fmt.Println(p.Name, p.Age)
}
func processPoint(p *Point) {
	fmt.Println(p.X, p.Y)
}
func processSlice(s *[]int) {
	fmt.Println(len(*s))
}
func processMap(m *map[string]int) {
	fmt.Println(len(*m))
}
func processArray(a *[3]int) {
	fmt.Println(len(*a))
}
// Test auto-reference for typed composite literals
func main() {
	processPerson(&Person{Name: "Alice", Age: 30})
	processPoint(&Point{X: 10, Y: 20})
	processSlice(&[]int{1, 2, 3})
	processMap(&map[string]int{"a": 1, "b": 2})
	processArray(&[3]int{1, 2, 3})
	processPerson(&Person{Name: "Bob", Age: 25})
	processPoint(&Point{X: 5, Y: 15})
}
