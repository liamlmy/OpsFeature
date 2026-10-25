package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/liamlmy/OpsFeature/feature"
	_ "github.com/liamlmy/OpsFeature/operator"
)

const (
	initialInputBufferSize = 64 * 1024
	maxInputLineSize       = 8 * 1024 * 1024
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run processes one input stream and returns the command's exit status.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "cmd args len wrong, args len is", len(args)+1)
		return 1
	}

	parser, err := feature.InitParser(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "init offline extractor error: %v\n", err)
		return 1
	}

	// Reset the per-sample state and reuse the FID slice's backing array.
	var (
		input = &feature.Input{}
		state = feature.NewState()
		fids  []feature.Fid
	)
	if parser.IsDebug == 1 {
		fmt.Fprintln(stderr, "debug is not supported, fall back to normal output")
	}

	scanner := bufio.NewScanner(stdin)
	scanner.Buffer(make([]byte, initialInputBufferSize), maxInputLineSize)
	for scanner.Scan() {
		fields, err := feature.ParseFeatureDataPerLineStrict(scanner.Text(), parser.FeatureNameList)
		if err != nil {
			fmt.Fprintf(stderr, "skip malformed input line: %v\n", err)
			continue
		}
		if !parser.ValidRequiredInput(fields) {
			fmt.Fprintln(stderr, "skip input line with empty required key fields")
			continue
		}

		input.Score = make(map[string]interface{}, len(fields))
		for name, value := range fields {
			input.Score[name] = value
		}
		state.Reset()
		fids, err = parser.OpExtractor.Extract(input, state, fids[:0])
		if err != nil {
			fmt.Fprintf(stderr, "operator Extract error: %v\n", err)
		}
		output, err := feature.OutputPerLineOp(fields, parser.LabelList, parser.AddTokenList, fids)
		if err != nil {
			fmt.Fprintf(stderr, "OutputPerLineOp error: %v\n", err)
			continue
		}
		fmt.Fprintln(stdout, output)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(stderr, "read stdin error: %v\n", err)
		return 1
	}
	return 0
}
