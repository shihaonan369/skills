package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"strings"
	"time"

	d "agent-dashboard/internal/dashboard"
)

func runScope(command string, args []string, out io.Writer) error {
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	f.SetOutput(out)
	path := f.String("file", "", "absolute private scope mailbox path")
	criterion := f.String("criterion", "", "caller-authored monitoring criterion")
	wait := f.Int("wait", 0, "bounded long poll in seconds, 0..60")
	duration := f.Int("duration", 300, "event stream lifetime in seconds, 1..1800")
	value := f.String("json", "", "decision batch JSON or @file")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *path == "" || f.NArg() != 0 {
		return errors.New("scope commands require --file and no positional arguments")
	}
	switch command {
	case "scope-init":
		return d.InitScope(*path, *criterion)
	case "scope-status":
		record, err := d.InspectScope(*path)
		if err != nil {
			return err
		}
		return json.NewEncoder(out).Encode(record)
	case "scope-close":
		return d.CloseScope(*path)
	case "scope-watch":
		if *duration < 1 || *duration > 1800 {
			return errors.New("--duration must be 1..1800 seconds")
		}
		encoder := json.NewEncoder(out)
		return d.WatchScope(context.Background(), *path, time.Duration(*duration)*time.Second, func(batch d.ScopeBatch) error { return encoder.Encode(batch) })
	case "scope-poll":
		if *wait < 0 || *wait > 60 {
			return errors.New("--wait must be 0..60 seconds")
		}
		batch, err := d.PollScope(*path, time.Duration(*wait)*time.Second)
		if err != nil {
			return err
		}
		return json.NewEncoder(out).Encode(batch)
	case "scope-decide":
		raw := []byte(*value)
		if strings.HasPrefix(*value, "@") {
			f, err := os.Open((*value)[1:])
			if err != nil {
				return err
			}
			defer f.Close()
			raw, err = io.ReadAll(io.LimitReader(f, (8<<20)+1))
			if err != nil {
				return err
			}
		}
		return d.DecideScope(*path, raw)
	}
	return errors.New("unknown scope command")
}
