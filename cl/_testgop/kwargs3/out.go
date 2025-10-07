package main

type Options map[string]bool

func PlaySound(options Options, paths ...string) {
}
func main() {
	PlaySound(Options{"loop": false}, "1.mp3", "foo.wav")
	PlaySound(Options{"loop": true, "async": true}, "2.mp3")
}
