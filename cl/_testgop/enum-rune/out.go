package main

import "fmt"

type Vowel int32

const (
	A Vowel = 'a'
	E Vowel = 'e'
	I Vowel = 'i'
	O Vowel = 'o'
	U Vowel = 'u'
)

func main() {
	fmt.Println(A, E, I)
}
