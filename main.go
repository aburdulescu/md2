package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"

	_ "embed"
)

//go:embed example.md
var example string

func main() {
	if err := mainErr(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func mainErr() error {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage: md2 [-h=header.html] [-f=footer.html] [-o=output] [FILE.md]

Convert the given FILE.md(or read stdin if not specified) to HTML.
Optionally, apply given header and footer to the generated HTML file.

Flags:
`)
		flag.PrintDefaults()
	}

	headerFile := flag.String("h", "", "Path to header file")
	footerFile := flag.String("f", "", "Path to footer file")
	outputFile := flag.String("o", "", "Path to output file")
	printVersion := flag.Bool("version", false, "Print version")
	printExample := flag.Bool("example", false, "Print example")
	flag.Parse()

	if *printVersion {
		bi, _ := debug.ReadBuildInfo()
		g := func(key string) string {
			for _, v := range bi.Settings {
				if v.Key == key {
					return v.Value
				}
			}
			return ""
		}
		fmt.Println(bi.Main.Version, bi.GoVersion, g("GOOS"), g("GOARCH"), g("vcs.revision"), g("vcs.time"))
		return nil
	}

	if *printExample {
		fmt.Print(example)
		return nil
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

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.DefinitionList,
			extension.Footnote,
		),
		goldmark.WithParserOptions(
			// useful for fragment links: href=#ID
			parser.WithAutoHeadingID(),
			parser.WithAttribute(),
		),
	)

	if err := md.Convert(input.Bytes(), &output); err != nil {
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
