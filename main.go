package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"

	_ "embed"
)

const usage = `Usage:

md2 -h
md2 -help

Print this message.

md2 -version

Print tool version.

md2 -example

Print a short tutorial with all supported Markdown features.

md2 [-head=header.html] [-foot=footer.html] [-o=output] [FILE.md]

Convert the given FILE.md(or read stdin if not specified) to HTML.
Optionally, apply given header and footer to the generated HTML file.

md2 [-head=header.html] [-foot=footer.html] [-o=output] DIRECTORY

Walk recursively the given DIRECTORY and copy all its files and
sub-directories to the output specified by -o.
Markdown files will be converted to HTML and the header and footer
will be applied to them, if provided.
All other files will be copied as is.

md2 -serve [-serve-addr=host:port] DIRECTORY

Serve DIRECTORY(or current directory if not specified) over HTTP.

Flags:
`

var (
	headerFile    = flag.String("head", "", "Path to header file")
	footerFile    = flag.String("foot", "", "Path to footer file")
	outputFile    = flag.String("o", "", "Path to output file/directory")
	shouldRecurse = flag.Bool("r", false, "Activate recursive walk of given directory")
	printVersion  = flag.Bool("version", false, "Print version")
	printExample  = flag.Bool("example", false, "Print example")
	serveFiles    = flag.Bool("serve", false, "Serve files")
	serveAddr     = flag.String("serve-addr", "localhost:12345", "Serve address")
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
		fmt.Fprint(os.Stderr, usage)
		flag.PrintDefaults()
	}
	flag.Parse()

	switch {
	case *printVersion:
		bi, _ := debug.ReadBuildInfo()
		g := func(key string) string {
			for _, v := range bi.Settings {
				if v.Key == key {
					return v.Value
				}
			}
			return ""
		}
		fmt.Println(
			bi.Main.Version, bi.GoVersion,
			g("GOOS"), g("GOARCH"),
			g("vcs.revision"), g("vcs.time"),
		)
		return nil

	case *printExample:
		fmt.Print(example)
		return nil

	case *serveFiles:
		dir := "."
		if flag.NArg() != 0 {
			dir = flag.Arg(0)
		}
		fileServer := http.FileServer(http.Dir(dir))
		http.Handle("/", fileServer)
		s := &http.Server{
			Addr:           *serveAddr,
			Handler:        nil,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
		fmt.Printf("serving files from %s on %s\n", dir, *serveAddr)
		return s.ListenAndServe()
	}

	if flag.NArg() > 1 {
		return fmt.Errorf("cannot handle more than one input")
	}

	if flag.NArg() == 0 {
		var w io.Writer = os.Stdout
		if *outputFile != "" {
			f, err := os.Create(*outputFile)
			if err != nil {
				return err
			}
			defer f.Close()
			w = f
		}
		return convert(os.Stdin, w, *headerFile, *footerFile)
	}

	input := flag.Arg(0)

	fi, err := os.Stat(input)
	if err != nil {
		return err
	}

	if fi.IsDir() {

		if !*shouldRecurse {
			return fmt.Errorf("-r must be specified if a directory is provided")
		}
		if *outputFile == "" {
			return fmt.Errorf("-o must be specified if a directory is provided")
		}
		walker := Walker{src: input, dst: *outputFile}
		return filepath.WalkDir(input, walker.walk)

	} else {

		f, err := os.Open(input)
		if err != nil {
			return err
		}
		defer f.Close()
		var w io.Writer = os.Stdout
		if *outputFile != "" {
			f, err := os.Create(*outputFile)
			if err != nil {
				return err
			}
			defer f.Close()
			w = f
		}
		return convert(f, w, *headerFile, *footerFile)

	}

}

func convert(r io.Reader, w io.Writer, headerFile, footerFile string) error {
	var input bytes.Buffer
	if _, err := io.Copy(&input, r); err != nil {
		return err
	}

	var output bytes.Buffer

	if headerFile != "" {
		header, err := os.Open(headerFile)
		if err != nil {
			return err
		}
		defer header.Close()
		if _, err := io.Copy(&output, header); err != nil {
			return err
		}
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
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)

	if err := md.Convert(input.Bytes(), &output); err != nil {
		return err
	}

	if footerFile != "" {
		footer, err := os.Open(footerFile)
		if err != nil {
			return err
		}
		defer footer.Close()
		if _, err := io.Copy(&output, footer); err != nil {
			return err
		}
	}

	if _, err := io.Copy(w, &output); err != nil {
		return err
	}

	return nil
}

type Walker struct {
	src string
	dst string
}

func (w Walker) walk(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if d.IsDir() {
		return w.mkdir(path)
	} else {
		return w.mkfile(path)
	}
}

func (w Walker) mkdir(path string) error {
	dir := filepath.Join(w.dst, strings.TrimPrefix(path, w.src))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return nil
}

func (w Walker) mkfile(path string) error {
	if path == *headerFile || path == *footerFile {
		return nil
	}
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()
	switch filepath.Ext(path) {
	case ".md":
		return w.mkfileMarkdown(src, path)
	default:
		return w.mkfileRegular(src, path)
	}
}

func (w Walker) mkfileRegular(src *os.File, path string) error {
	dst, err := os.Create(filepath.Join(w.dst, strings.TrimPrefix(path, w.src)))
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	if err := dst.Sync(); err != nil {
		return err
	}
	return nil
}

func (w Walker) mkfileMarkdown(src *os.File, path string) error {
	dir, file := filepath.Split(strings.TrimPrefix(path, w.src))
	dstpath := filepath.Join(w.dst, dir, changeFileExt(file, ".html"))
	dst, err := os.Create(dstpath)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	defer dst.Close()
	return convert(src, dst, *headerFile, *footerFile)
}

func changeFileExt(filename, newExt string) string {
	dots := strings.Split(filename, ".")
	if len(dots) == 0 {
		return filename
	}
	newfilename := strings.Join(dots[:len(dots)-1], ".")
	newfilename += newExt
	return newfilename
}
