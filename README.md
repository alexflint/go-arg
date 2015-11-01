Argument parsing for Go.

```golang
var args struct {
	Foo  string
	Bar  bool
}
arg.MustParse(&args)
fmt.Println(args.Foo)
fmt.Println(args.Bar)
```

```bash
$ ./example --foo=hello --bar
hello
True
```

Setting defaults values:

```golang
var args struct {
	Foo string
	Bar bool
}
args.Foo = "default value"
arg.MustParse(&args)
```

Marking options as required

```golang
var args struct {
	Foo string `arg:"required"`
	Bar bool
}
arg.MustParse(&args)
```

Positional argument:

```golang
var args struct {
	Input   string   `arg:"positional"`
	Output  []string `arg:"positional"`
	Verbose bool
}
arg.MustParse(&args)
fmt.Println("Input:", input)
fmt.Println("Output:", output)
```

```
$ ./example src.txt x.out y.out z.out
Input: src.txt
Output: [x.out y.out z.out]
```

Usage strings:
```bash
$ ./example -h
usage: [--verbose] [--dataset DATASET] [--optimize OPTIMIZE] [--help] INPUT [OUTPUT [OUTPUT ...]] 

positional arguments:
  input
  output

options:
--verbose, -v            verbosity level
--dataset DATASET        dataset to use
--optimize OPTIMIZE, -O OPTIMIZE
                         optimization level
--help, -h               print this help message
```

Options with multiple values:
```
var args struct {
	Database string
	IDs      []int64
}
arg.MustParse(&args)
fmt.Printf("Fetching the following IDs from %s: %q", args.Database, args.IDs)
```

```bash
./example -database foo -ids 1 2 3
Fetching the following IDs from foo: [1 2 3]
```
