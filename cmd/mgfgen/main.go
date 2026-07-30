// Command mgfgen creates ModGadget Font files.
//
// Without -bdf it writes an empty MGF1 header. With -bdf it converts a BDF
// font to an uncompressed MGF1 file.
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
	bdfPath := flags.String("bdf", "", "input BDF 2.1 or 2.2 font")
	charsPath := flags.String("chars", "", "UTF-8 file selecting characters from the BDF")
	missing := flags.String("missing", "error", "missing selected characters: error or skip")
	assumeUnicode := flags.Bool("assume-unicode", false, "treat an unrecognized BDF charset as Unicode")
	lineGap := flags.Int("line-gap", 0, "MGF line gap from 0 through 255")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "mgfgen creates an uncompressed MGF1 file from BDF, or a header-only empty MGF1 without -bdf.")
		fmt.Fprintln(stderr, "BDF conversion writes raw 1-bit glyph records; omit -bdf for the header-only mode.")
		fmt.Fprintln(stderr, "Usage: mgfgen -bdf font.bdf -font-id sh12 -subset-id full -region JP -o font.mgf")
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
	if *missing != "error" && *missing != "skip" {
		fmt.Fprintln(stderr, "mgfgen: -missing must be error or skip")
		return 2
	}
	if *lineGap < 0 || *lineGap > 255 {
		fmt.Fprintln(stderr, "mgfgen: -line-gap must be from 0 through 255")
		return 2
	}
	if *bdfPath == "" && (*charsPath != "" || *missing != "error" || *assumeUnicode || *lineGap != 0) {
		fmt.Fprintln(stderr, "mgfgen: BDF conversion flags require -bdf")
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
	if *bdfPath != "" {
		options := conversionOptions{
			bdfPath:       *bdfPath,
			charsPath:     *charsPath,
			missing:       *missing,
			assumeUnicode: *assumeUnicode,
			lineGap:       uint8(*lineGap),
			fontID:        fontBytes,
			subsetID:      subsetBytes,
			region:        regionBytes,
			output:        *output,
		}
		if err := convertBDF(options, stdout, stderr); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
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
