// +build ignore

package main

import (
	"flag"
	"time"

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
	ts, err := internal.NewTimestampFileWriter(timestampFileFlag)
	if err != nil {
		panic(err)
	}
	ts.Append(time.Now())
	defer ts.Close()
}
