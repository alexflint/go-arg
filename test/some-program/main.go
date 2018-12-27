package main

import "github.com/alexflint/go-arg"

func main() {
	var args struct {
		Test string
	}
	arg.MustParse(&args)
}
