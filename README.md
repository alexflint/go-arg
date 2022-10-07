<h1 align="center">
  <img src="./.github/banner.jpg" alt="go-arg" height="250px">
  <br>
  go-arg
  </br>
</h1>
<h4 align="center">Struct-based argument parsing for Go</h4>
<p align="center">
  <a href="https://sourcegraph.com/github.com/alexflint/go-arg?badge"><img src="https://sourcegraph.com/github.com/alexflint/go-arg/-/badge.svg" alt="Sourcegraph"></a>
  <a href="https://pkg.go.dev/github.com/alexflint/go-arg"><img src="https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square" alt="Documentation"></a>
  <a href="https://github.com/alexflint/go-arg/actions"><img src="https://github.com/alexflint/go-arg/workflows/Go/badge.svg" alt="Build Status"></a>
  <a href="https://codecov.io/gh/alexflint/go-arg"><img src="https://codecov.io/gh/alexflint/go-arg/branch/master/graph/badge.svg" alt="Coverage Status"></a>
  <a href="https://goreportcard.com/report/github.com/alexflint/go-arg"><img src="https://goreportcard.com/badge/github.com/alexflint/go-arg" alt="Go Report Card"></a>
</p>
<br>

Declare command line arguments for your program by defining a struct.

```go
import "github.com/go-arg/v2
```

TODO

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
hello true
```

### Installation

```shell
go get github.com/alexflint/go-arg/v2
```

### Required arguments

```go
var args struct {
	ID      int `arg:"required"`
	Timeout time.Duration
}
arg.MustParse(&args)
```

```shell
$ ./example
Usage: example --id ID [--timeout TIMEOUT]
error: --id is required
```

### Positional arguments

```go
var args struct {
	Input   string   `arg:"positional"`
	Output  []string `arg:"positional"`
}
arg.MustParse(&args)
fmt.Println("Input:", args.Input)
fmt.Println("Output:", args.Output)
```

```
$ ./example src.txt x.out y.out z.out
Input: src.txt
Output: [x.out y.out z.out]
```

### Environment variables

```go
var args struct {
	Workers int `arg:"env"`
}
arg.MustParse(&args)
fmt.Println("Workers:", args.Workers)
```

```
$ WORKERS=4 ./example
Workers: 4
```

```
$ WORKERS=4 ./example --workers=6
Workers: 6
```

### Usage strings
```go
var args struct {
	Input    string   `arg:"positional"`
	Output   []string `arg:"positional"`
	Verbose  bool     `arg:"-v,--verbose" help:"verbosity level"`
	Dataset  string   `help:"dataset to use"`
	Optimize int      `arg:"-O" help:"optimization level"`
}
arg.MustParse(&args)
```

```shell
$ ./example -h
Usage: [--verbose] [--dataset DATASET] [--optimize OPTIMIZE] [--help] INPUT [OUTPUT [OUTPUT ...]]

Positional arguments:
  INPUT
  OUTPUT

Options:
  --verbose, -v            verbosity level
  --dataset DATASET        dataset to use
  --optimize OPTIMIZE, -O OPTIMIZE
                           optimization level
  --help, -h               print this help message
```

### Default values

```go
var args struct {
	Foo string `default:"abc"`
	Bar bool
}
arg.MustParse(&args)
```

### Overriding the name of an environment variable

```go
var args struct {
	Workers int `arg:"env:NUM_WORKERS"`
}
arg.MustParse(&args)
fmt.Println("Workers:", args.Workers)
```

```
$ NUM_WORKERS=4 ./example
Workers: 4
```

### Arguments with multiple values

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

### Arguments with keys and values
```go
var args struct {
	UserIDs map[string]int
}
arg.MustParse(&args)
fmt.Println(args.UserIDs)
```

```shell
./example --userids john=123 mary=456
map[john:123 mary:456]
```

### Custom validation
```go
var args struct {
	Foo string
	Bar string
}
p := arg.MustParse(&args)
if args.Foo == "" && args.Bar == "" {
	p.Fail("you must provide either --foo or --bar")
}
```

```shell
./example
Usage: samples [--foo FOO] [--bar BAR]
error: you must provide either --foo or --bar
```

### Version strings

```go
type args struct {
	// ...
}

func (args) Version() string {
	return "someprogram 4.3.0"
}

func main() {
	var args args
	arg.MustParse(&args)
}
```

```shell
$ ./example --version
someprogram 4.3.0
```

### Overriding option names

```go
var args struct {
	Short        string `arg:"-s"`
	Long         string `arg:"--custom-long-option"`
	ShortAndLong string `arg:"-x,--my-option"`
	OnlyShort    string `arg:"-o,--"`
}
arg.MustParse(&args)
```

```shell
$ ./example --help
Usage: example [-o ONLYSHORT] [--short SHORT] [--custom-long-option CUSTOM-LONG-OPTION] [--my-option MY-OPTION]

Options:
  --short SHORT, -s SHORT
  --custom-long-option CUSTOM-LONG-OPTION
  --my-option MY-OPTION, -x MY-OPTION
  -o ONLYSHORT
  --help, -h             display this help and exit
```


### Embedded structs

The fields of embedded structs are treated just like regular fields:

```go

type DatabaseOptions struct {
	Host     string
	Username string
	Password string
}

type LogOptions struct {
	LogFile string
	Verbose bool
}

func main() {
	var args struct {
		DatabaseOptions
		LogOptions
	}
	arg.MustParse(&args)
}
```

As usual, any field tagged with `arg:"-"` is ignored.

### Supported types

The following types may be used as arguments:
- built-in integer types: `int, int8, int16, int32, int64, byte, rune`
- built-in floating point types: `float32, float64`
- strings
- booleans
- URLs represented as `url.URL`
- time durations represented as `time.Duration`
- email addresses represented as `mail.Address`
- MAC addresses represented as `net.HardwareAddr`
- pointers to any of the above
- slices of any of the above
- maps using any of the above as keys and values
- any type that implements `encoding.TextUnmarshaler`

### Custom parsing 

Implement `encoding.TextUnmarshaler` to define your own parsing logic.

```go
// Accepts command line arguments of the form "head.tail"
type NameDotName struct {
	Head, Tail string
}

func (n *NameDotName) UnmarshalText(b []byte) error {
	s := string(b)
	pos := strings.Index(s, ".")
	if pos == -1 {
		return fmt.Errorf("missing period in %s", s)
	}
	n.Head = s[:pos]
	n.Tail = s[pos+1:]
	return nil
}

func main() {
	var args struct {
		Name NameDotName
	}
	arg.MustParse(&args)
	fmt.Printf("%#v\n", args.Name)
}
```
```shell
$ ./example --name=foo.bar
main.NameDotName{Head:"foo", Tail:"bar"}

$ ./example --name=oops
Usage: example [--name NAME]
error: error processing --name: missing period in "oops"
```

### Slice-valued environment variables

You can provide multiple values using the CSV (RFC 4180) format:

```go
var args struct {
    Workers []int `arg:"env"`
}
arg.MustParse(&args)
fmt.Println("Workers:", args.Workers)
```

```
$ WORKERS='1,99' ./example
Workers: [1 99]
```

### Parsing command line tokens and environment variables from a slice

You can override the command line tokens and environment variables processed by go-arg:

```go
var args struct {
	Samsara int
	Nirvana float64 `arg:"env:NIRVANA"`
}
p, err := arg.NewParser(&args)
if err != nil {
	log.Fatal(err)
}
cmdline := []string{"./thisprogram", "--samsara=123"}
environ := []string{"NIRVANA=45.6"}
err = p.Parse(cmdline, environ)
if err != nil {
	log.Fatal(err)
}
```
```
./example
SAMSARA: 123
NIRVANA: 45.6
```

### Configuration files

TODO

### Combining command line options, environment variables, and default values

By default, command line arguments take precedence over environment variables, which take precedence over default values. This means that we check whether a certain option was provided on the command line, then if not, we check for an environment variable (only if an `env` tag was provided), then, if none is found, we check for a `default` tag.

```go
var args struct {
    Test  string `arg:"-t,env:TEST" default:"something"`
}
arg.MustParse(&args)
```

### Changing precedence of command line options, environment variables, and default values

You can use the low-level functions `Process*` and `OverwriteWith*` to control which things override which other things. Here is an example in which environment variables take precedence over command line options, which is the opposite of the default behavior:

```go
var args struct {
	Test string `arg:"env:TEST"`
}

p, err := arg.NewParser(&args)
if err != nil {
	log.Fatal(err)
}

err = p.ParseCommandLine(os.Args)
if err != nil {
	p.Fail(err.Error())
}

err = p.OverwriteWithEnvironment(os.Environ())
if err != nil {
	p.Fail(err.Error())
}

err = p.Validate()
if err != nil {
	p.Fail(err.Error())
}

fmt.Printf("test=%s\n", args.Test)
```
```
TEST=value_from_env ./example --test=value_from_option
test=value_from_env
```

### Ignoring environment variables

TODO

### Ignoring default values

TODO

### Arguments that can be specified multiple times
```go
var args struct {
    Commands  []string `arg:"-c,separate"`
    Files     []string `arg:"-f,separate"`
}
arg.MustParse(&args)
```

```shell
./example -c cmd1 -f file1 -c cmd2 -f file2 -f file3 -c cmd3
Commands: [cmd1 cmd2 cmd3]
Files [file1 file2 file3]
```

### Description strings

A descriptive message can be added at the top of the help text by implementing
a `Description` function that returns a string.

```go
type args struct {
	Foo string
}

func (args) Description() string {
	return "this program does this and that"
}

func main() {
	var args args
	arg.MustParse(&args)
}
```

```shell
$ ./example -h
this program does this and that
Usage: example [--foo FOO]

Options:
  --foo FOO
  --help, -h             display this help and exit
```

Similarly an epilogue can be added at the end of the help text by implementing
the `Epilogue` function.

```go
type args struct {
	Foo string
}

func (args) Epilogue() string {
	return "For more information visit github.com/alexflint/go-arg"
}

func main() {
	var args args
	arg.MustParse(&args)
}
```

```shell
$ ./example -h
Usage: example [--foo FOO]

Options:
  --foo FOO
  --help, -h             display this help and exit

For more information visit github.com/alexflint/go-arg
```

### Subcommands

Subcommands are commonly used in tools that group multiple functions into a single program. An example is the `git` tool:
```shell
$ git checkout [arguments specific to checking out code]
$ git commit [arguments specific to committing code]
$ git push [arguments specific to pushing code]
```

This can be implemented with `go-arg` with the `arg:"subcommand"` tag:

```go
type CheckoutCmd struct {
	Branch string `arg:"positional"`
	Track  bool   `arg:"-t"`
}
type CommitCmd struct {
	All     bool   `arg:"-a"`
	Message string `arg:"-m"`
}
type PushCmd struct {
	Remote      string `arg:"positional"`
	Branch      string `arg:"positional"`
	SetUpstream bool   `arg:"-u"`
}
var args struct {
	Checkout *CheckoutCmd `arg:"subcommand:checkout"`
	Commit   *CommitCmd   `arg:"subcommand:commit"`
	Push     *PushCmd     `arg:"subcommand:push"`
	Quiet    bool         `arg:"-q"` // this flag is global to all subcommands
}

arg.MustParse(&args)

switch {
case args.Checkout != nil:
	fmt.Printf("checkout requested for branch %s\n", args.Checkout.Branch)
case args.Commit != nil:
	fmt.Printf("commit requested with message \"%s\"\n", args.Commit.Message)
case args.Push != nil:
	fmt.Printf("push requested from %s to %s\n", args.Push.Branch, args.Push.Remote)
}
```

Note that the `subcommand` tag can only be used with fields that are pointers to structs, and that any struct that contains subcommands cannot also contain positionals.

### Terminating when no subcommands are specified

```go
p := arg.MustParse(&args)
if p.Subcommand() == nil {
    p.Fail("missing subcommand")
}
```

### Customizing placeholder strings

Use the `placeholder` tag to control which placeholder text is used in the usage text.

```go
var args struct {
	Input    string   `arg:"positional" placeholder:"SRC"`
	Output   []string `arg:"positional" placeholder:"DST"`
	Optimize int      `arg:"-O" placeholder:"LEVEL"`
	MaxJobs  int      `arg:"-j" placeholder:"N"`
}
arg.MustParse(&args)
```
```shell
$ ./example -h
Usage: example [--optimize LEVEL] [--maxjobs N] SRC [DST [DST ...]]

Positional arguments:
  SRC
  DST

Options:
  --optimize LEVEL, -O LEVEL
  --maxjobs N, -j N
  --help, -h             display this help and exit
```

### API Documentation

https://godoc.org/github.com/alexflint/go-arg

### Migrating from v1.x

Migrating IgnoreEnv to passing a nil environ

Migrating from IgnoreDefault to calling ProcessCommandLine