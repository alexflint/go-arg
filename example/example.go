package main

import "github.com/alexflint/go-arg"

func main() {
	var args struct {
		Input    string   `arg:"positional"`
		Output   []string `arg:"positional"`
		Verbose  bool     `arg:"-v,help:verbosity level"`
		Dataset  string   `arg:"help:dataset to use"`
		Optimize int      `arg:"-O,help:optimization level"`
	}
	arg.MustParse(&args)
}
