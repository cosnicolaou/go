// +build ignore

package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/cosnicolaou/go/cmd/pulsemon/internal"
)

var (
	timestampFileFlag string
	fromFlag, toFlag  string
)

func init() {
	flag.StringVar(&timestampFileFlag, "timestamp-file", "", "file containing timestamps")
	flag.StringVar(&fromFlag, "from", "", "start of time period")
	flag.StringVar(&toFlag, "to", "", "end of time period")
}

func main() {
	flag.Parse()
	var from, to time.Time
	var err error
	if len(fromFlag) > 0 {
		from, err = time.Parse("2006-01-02 15:04:05", fromFlag)
		if err != nil {
			panic(fmt.Sprintf("failed to parse %v", fromFlag))
		}
	}
	if len(toFlag) > 0 {
		to, err = time.Parse("2006-01-02 15:04:05", toFlag)
		if err != nil {
			panic(fmt.Sprintf("failed to parse %v", toFlag))
		}
	} else {
		to = time.Now()
	}
	if err := internal.ReadTimestamps(timestampFileFlag, from, to); err != nil {
		panic(err)
	}
}
