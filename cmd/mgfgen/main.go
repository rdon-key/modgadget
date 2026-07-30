// Command mgfgen creates ModGadget Font files.
//
// This first stage writes only an empty MGF1 header. BDF conversion and glyph
// generation are intentionally not implemented yet.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rdon-key/modgadget/internal/mgf"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("mgfgen", flag.ContinueOnError)
	flags.SetOutput(stderr)
	fontID := flags.String("font-id", "", "four printable ASCII bytes identifying the font")
	subsetID := flags.String("subset-id", "", "four printable ASCII bytes identifying the subset")
	region := flags.String("region", "", "two printable ASCII bytes, or empty for no region")
	output := flags.String("o", "", "output MGF file (required)")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "mgfgen creates a header-only empty MGF1 file.")
		fmt.Fprintln(stderr, "BDF conversion and glyph generation are not implemented yet.")
		fmt.Fprintln(stderr, "Usage: mgfgen -font-id sh12 -subset-id full -region JP -o empty.mgf")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "mgfgen: unexpected arguments: %v\n", flags.Args())
		return 2
	}
	if *fontID == "" {
		fmt.Fprintln(stderr, "mgfgen: -font-id is required")
		return 2
	}
	if *subsetID == "" {
		fmt.Fprintln(stderr, "mgfgen: -subset-id is required")
		return 2
	}
	if *output == "" {
		fmt.Fprintln(stderr, "mgfgen: -o is required")
		return 2
	}

	fontBytes, err := fixedPrintableASCII(*fontID, 4, "font-id")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	subsetBytes, err := fixedPrintableASCII(*subsetID, 4, "subset-id")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	var regionBytes []byte
	if *region != "" {
		regionBytes, err = fixedPrintableASCII(*region, 2, "region")
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
	}

	header := mgf.Header{
		Version:         mgf.Version1,
		HeaderSize:      mgf.HeaderSize,
		IndexOffset:     mgf.HeaderSize,
		GlyphDataOffset: mgf.HeaderSize,
		FileSize:        mgf.HeaderSize,
	}
	copy(header.FontID[:], fontBytes)
	copy(header.SubsetID[:], subsetBytes)
	copy(header.Region[:], regionBytes)
	var encoded [mgf.HeaderSize]byte
	if err := mgf.EncodeHeader(encoded[:], header); err != nil {
		fmt.Fprintf(stderr, "mgfgen: encode header: %v\n", err)
		return 1
	}
	if err := writeAtomic(*output, encoded[:]); err != nil {
		fmt.Fprintf(stderr, "mgfgen: write %s: %v\n", *output, err)
		return 1
	}

	fmt.Fprintf(stdout, "wrote %s\n", *output)
	fmt.Fprintf(stdout, "FontID: %s\n", *fontID)
	fmt.Fprintf(stdout, "SubsetID: %s\n", *subsetID)
	if *region == "" {
		fmt.Fprintln(stdout, "Region: none")
	} else {
		fmt.Fprintf(stdout, "Region: %s\n", *region)
	}
	fmt.Fprintln(stdout, "GlyphCount: 0")
	fmt.Fprintf(stdout, "bytes: %d\n", mgf.HeaderSize)
	return 0
}

func fixedPrintableASCII(value string, size int, name string) ([]byte, error) {
	if len(value) != size {
		return nil, fmt.Errorf("mgfgen: -%s must be exactly %d bytes", name, size)
	}
	for index := 0; index < len(value); index++ {
		if value[index] < 0x20 || value[index] > 0x7e {
			return nil, fmt.Errorf("mgfgen: -%s must contain printable ASCII only", name)
		}
	}
	return []byte(value), nil
}

func writeAtomic(path string, data []byte) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	closed := false
	defer func() {
		if !closed {
			_ = temporary.Close()
		}
		_ = os.Remove(temporaryName)
	}()

	written, err := temporary.Write(data)
	if err != nil {
		return err
	}
	if written != len(data) {
		return io.ErrShortWrite
	}
	if err := temporary.Close(); err != nil {
		closed = true
		return err
	}
	closed = true
	if err := os.Rename(temporaryName, path); err != nil {
		return err
	}
	return nil
}
