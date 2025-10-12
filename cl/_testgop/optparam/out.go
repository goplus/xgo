package main

import "fmt"

type Server struct {
}

func basic(a int, __xgo_optional_b int) {
	fmt.Println(a, __xgo_optional_b)
}
func multiple(name string, __xgo_optional_age int, __xgo_optional_active bool) {
	fmt.Println(name, __xgo_optional_age, __xgo_optional_active)
}
func allOptional(__xgo_optional_x int, __xgo_optional_y string) {
	fmt.Println(__xgo_optional_x, __xgo_optional_y)
}
func withVariadic(__xgo_optional_a int, b ...string) {
	fmt.Println(__xgo_optional_a, b)
}
func (s *Server) handle(req string, __xgo_optional_opts int) {
	fmt.Println(req, __xgo_optional_opts)
}
func main() {
	basic(10, 20)
	multiple("Alice", 30, true)
	allOptional(100, "test")
	withVariadic(5, "hello", "world")
	s := Server{}
	s.handle("request", 42)
}
