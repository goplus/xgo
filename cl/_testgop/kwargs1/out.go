package main

type Options struct {
	Loop  bool
	async bool
}

func PlaySound(path string, options *Options) {
}
func main() {
	PlaySound("1.mp3", &Options{Loop: true})
	PlaySound("2.mp3", &Options{Loop: false, async: true})
}
