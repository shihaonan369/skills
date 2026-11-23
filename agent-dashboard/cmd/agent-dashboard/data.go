package main

import (
	"errors"
	"flag"
	"io"
	"os"
	"strings"

	d "agent-dashboard/internal/dashboard"
)

func runDataWrite(args []string, in io.Reader, out io.Writer) error {
	f := flag.NewFlagSet("data-write", flag.ContinueOnError)
	f.SetOutput(out)
	path := f.String("file", "", "absolute caller-owned JSON data path")
	value := f.String("json", "", "complete JSON value, @file, or - for stdin")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *path == "" || *value == "" || f.NArg() != 0 {
		return errors.New("data-write requires --file and --json with no positional arguments")
	}
	var reader io.Reader = strings.NewReader(*value)
	if *value == "-" {
		reader = in
	} else if strings.HasPrefix(*value, "@") {
		file, err := os.Open((*value)[1:])
		if err != nil {
			return err
		}
		defer file.Close()
		reader = file
	}
	raw, err := io.ReadAll(io.LimitReader(reader, d.MaxDataFileBytes+1))
	if err != nil {
		return err
	}
	return d.PublishDataFile(*path, raw)
}
