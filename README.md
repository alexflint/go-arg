# Argument parsing for Go

```go
var args struct {
	Foo string
	Bar bool
}
arg.MustParse(&args)
fmt.Println(args.Foo, args.Bar)
```

```shell
$ ./example --foo=hello --bar
hello True
```

### Default values

```go
var args struct {
	Foo string
	Bar bool
}
args.Foo = "default value"
arg.MustParse(&args)
```

### Marking options as required

```go
var args struct {
	Foo string `arg:"required"`
	Bar bool
}
arg.MustParse(&args)
```

### Positional argument

```go
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

### Usage strings
```go
var args struct {
	Input    string   `arg:"positional"`
	Output   []string `arg:"positional"`
	Verbose  bool     `arg:"-v,help:verbosity level"`
	Dataset  string   `arg:"help:dataset to use"`
	Optimize int      `arg:"-O,help:optimization level"`
}
arg.MustParse(&args)
```

```shell
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

### Options with multiple values
```go
var args struct {
	Database string
	IDs      []int64
}
arg.MustParse(&args)
fmt.Printf("Fetching the following IDs from %s: %q", args.Database, args.IDs)
```

```shell
./example -database foo -ids 1 2 3
Fetching the following IDs from foo: [1 2 3]
```
