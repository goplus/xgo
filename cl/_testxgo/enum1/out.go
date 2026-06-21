package main

import "fmt"

type Scale float64

const (
	Tiny   Scale = 0.25
	Small  Scale = 0.5
	Normal Scale = 1.0
	Large  Scale = 2.0
)

type LogLevel string

const (
	Debug LogLevel = "debug"
	Info  LogLevel = "info"
	Warn  LogLevel = "warn"
	Error LogLevel = "error"
)

type Vowel rune

const (
	A Vowel = 'a'
	E Vowel = 'e'
	I Vowel = 'i'
	O Vowel = 'o'
	U Vowel = 'u'
)

type Toggle bool

const (
	Off Toggle = false
	On  Toggle = true
)

func main() {
	fmt.Println(Tiny, Small, Normal, Large)
	fmt.Println(Debug, Info, Warn, Error)
	fmt.Println(A, E, I)
	fmt.Println(Off, On)
}
