package main

import "github.com/alexflint/go-arg"

func main() {
	var args struct {
		Input            string  `arg:"positional"`
		Output           string  `arg:"positional"`
		Foo              string  `arg:"help:this argument is foo"`
		VeryLongArgument int     `arg:"help:this argument is very long"`
		Bar              float64 `arg:"-b"`
	}
	arg.MustParse(&args)
}
