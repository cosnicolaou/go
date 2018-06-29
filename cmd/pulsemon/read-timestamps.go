// +build ignore

package main

import (
	"flag"

	"github.com/cosnicolaou/go/cmd/pulsemon/internal"
)

var (
	timestampFileFlag string
)

func init() {
	flag.StringVar(&timestampFileFlag, "timestamp-file", "", "file containing timestamps")
}

func main() {
	flag.Parse()
	if err := internal.ReadTimestamps(timestampFileFlag); err != nil {
		panic(err)
	}
}
