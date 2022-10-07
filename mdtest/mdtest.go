// mdtest executes code blocks in markdown and checks that they run as expected
package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/alexflint/go-arg/v2"
)

// var pattern = "```go(.*)```\\s*```\\s*\\$(.*)\\n(.*)```"
var pattern = "(?s)```go([^`]*?)```\\s*```([^`]*?)```" //go(.*)```\\s*```\\s*\\$(.*)\\n(.*)```"

var re = regexp.MustCompile(pattern)

var funcs = map[string]any{
	"contains": strings.Contains,
}

//go:embed example1.go.tpl
var templateSource1 string

//go:embed example2.go.tpl
var templateSource2 string

var t1 = template.Must(template.New("example1.go").Funcs(funcs).Parse(templateSource1))
var t2 = template.Must(template.New("example2.go").Funcs(funcs).Parse(templateSource2))

type payload struct {
	Code string
}

func runCode(ctx context.Context, code []byte, cmd string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, fmt.Errorf("error creating temp dir to build and run code: %w", err)
	}

	fmt.Println(dir)
	fmt.Println(strings.Repeat("-", 80))

	srcpath := filepath.Join(dir, "src.go")
	binpath := filepath.Join(dir, "example")

	// If the code contains a main function then use t2, otherwise use t1
	t := t1
	if strings.Contains(string(code), "func main") {
		t = t2
	}

	var b bytes.Buffer
	err = t.Execute(&b, payload{Code: string(code)})
	if err != nil {
		return nil, fmt.Errorf("error executing template for source file: %w", err)
	}

	fmt.Println(b.String())
	fmt.Println(strings.Repeat("-", 80))

	err = os.WriteFile(srcpath, b.Bytes(), os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("error writing temporary source file: %w", err)
	}

	compiler, err := exec.LookPath("go")
	if err != nil {
		return nil, fmt.Errorf("could not find path to go compiler: %w", err)
	}

	buildCmd := exec.CommandContext(ctx, compiler, "build", "-o", binpath, srcpath)
	out, err := buildCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error building source: %w. Compiler said:\n%s", err, string(out))
	}

	// replace "./example" with full path to compiled program
	var env, args []string
	var found bool
	for _, part := range strings.Split(cmd, " ") {
		if found {
			args = append(args, part)
		} else if part == "./example" {
			found = true
		} else {
			env = append(env, part)
		}
	}

	runCmd := exec.CommandContext(ctx, binpath, args...)
	runCmd.Env = env
	output, err := runCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error runing example: %w. Program said:\n%s", err, string(output))
	}

	// Clean up the temp dir
	if err := os.RemoveAll(dir); err != nil {
		return nil, fmt.Errorf("error deleting temp dir: %w", err)
	}

	return output, nil
}

func Main() error {
	ctx := context.Background()

	var args struct {
		Input string `arg:"positional,required"`
	}
	arg.MustParse(&args)

	buf, err := os.ReadFile(args.Input)
	if err != nil {
		return err
	}

	fmt.Println(strings.Repeat("=", 80))

	matches := re.FindAllSubmatchIndex(buf, -1)
	for k, match := range matches {
		codebegin, codeend := match[2], match[3]
		code := buf[codebegin:codeend]

		shellbegin, shellend := match[4], match[5]
		shell := buf[shellbegin:shellend]

		lines := strings.Split(string(shell), "\n")
		for i := 0; i < len(lines); i++ {
			if strings.HasPrefix(lines[i], "$") && strings.Contains(lines[i], "./example") {
				cmd := strings.TrimSpace(strings.TrimPrefix(lines[i], "$"))

				var output []string
				i++
				for i < len(lines) && !strings.HasPrefix(lines[i], "$") {
					output = append(output, lines[i])
					i++
				}

				expected := strings.TrimSpace(strings.Join(output, "\n"))

				fmt.Println(string(code))
				fmt.Println(strings.Repeat("-", 80))
				fmt.Println(string(cmd))
				fmt.Println(strings.Repeat("-", 80))
				fmt.Println(string(expected))
				fmt.Println(strings.Repeat("-", 80))

				ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()

				actual, err := runCode(ctx, code, cmd)
				if err != nil {
					return fmt.Errorf("error running example %d: %w\nCode was:\n%s", k, err, string(code))
				}

				fmt.Println(string(actual))
				fmt.Println(strings.Repeat("=", 80))
			}
		}
	}
	fmt.Printf("found %d matches\n", len(matches))
	return nil
}

func main() {
	if err := Main(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
