package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/yuin/goldmark"
)

func main() {
	if err := mainErr(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func mainErr() error {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage: md2 [-head header.html] [-foot footer.html] [FILE.md]

Convert the given FILE.md(or read stdin if not specified) to HTML.
Optionally, apply given header and footer to the generated HTML file.

Flags:
`)
		flag.PrintDefaults()
	}

	recurse := flag.Bool("r", false, "Recurse directories")
	headerFile := flag.String("head", "", "Path to header file")
	footerFile := flag.String("foot", "", "Path to footer file")
	outputFile := flag.String("o", "", "Path to output file")
	flag.Parse()

	if *recurse {
		return fmt.Errorf("not implemented")
	}

	if flag.NArg() > 1 {
		return fmt.Errorf("cannot handle more than one input file")
	}

	var r io.Reader = os.Stdin
	if flag.NArg() > 0 {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	}

	var w io.Writer = os.Stdout
	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}

	var input bytes.Buffer
	if _, err := io.Copy(&input, r); err != nil {
		return err
	}

	var output bytes.Buffer

	if *headerFile != "" {
		header, err := os.Open(*headerFile)
		if err != nil {
			return err
		}
		defer header.Close()
		io.Copy(&output, header)
	}

	if err := goldmark.Convert(input.Bytes(), &output); err != nil {
		return err
	}

	if *footerFile != "" {
		footer, err := os.Open(*footerFile)
		if err != nil {
			return err
		}
		defer footer.Close()
		io.Copy(&output, footer)
	}

	io.Copy(w, &output)

	return nil
}
