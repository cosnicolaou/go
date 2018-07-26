package internal

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

// TimestampFileWriter represents a binary file containing the encoded
// timestamps of each pulse received.
type TimestampFileWriter struct {
	io.WriteCloser
	name string
	buf  []byte
}

// NewTimestampFileWriter opens or creates a timestamp log file.
func NewTimestampFileWriter(filename string) (*TimestampFileWriter, error) {
	wr, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open %v: %v", filename, err)
	}
	return &TimestampFileWriter{WriteCloser: wr, name: filename, buf: make([]byte, 8)}, nil
}

// Append appends a time stamp to the underlying file.
func (tf *TimestampFileWriter) Append(ts time.Time) error {
	binary.LittleEndian.PutUint64(tf.buf, uint64(ts.UnixNano()))
	if _, err := tf.Write(tf.buf); err != nil {
		return fmt.Errorf("failed writing/appending to timestamp file %v: %v", tf.name, err)
	}
	return nil
}

// TimestampFileScanner represents a scanner for a timestamp file.
type TimestampFileScanner struct {
	ts  int64
	rd  io.Reader
	err error
}

// NewTimestampFileScanner creates a new TimestampFileScanner.
func NewTimestampFileScanner(rd io.Reader) *TimestampFileScanner {
	return &TimestampFileScanner{rd: rd}
}

// Err is analagous to bufio.Scanner.Err.
func (ts *TimestampFileScanner) Err() error {
	if ts.err == nil || ts.err == io.EOF {
		return nil
	}
	return ts.err
}

// Scan is analagous to bufio.Scanner.Scan.
func (ts *TimestampFileScanner) Scan() bool {
	if ts.err != nil {
		return false
	}
	ts.err = binary.Read(ts.rd, binary.LittleEndian, &ts.ts)
	return ts.err == nil
}

// Time is analogous to bufio.Scanner.Bytes.
func (ts *TimestampFileScanner) Time() time.Time {
	return time.Unix(0, ts.ts)
}

// ReadTimestamps read and print the timestamps.
func ReadTimestamps(filename string, from, to time.Time) error {
	var rd *os.File
	var err error
	if filename == "-" {
		rd = os.Stdin
	} else {
		rd, err = os.Open(filename)
		if err != nil {
			return err
		}
	}
	defer rd.Close()
	pulseCounter := 0
	fmt.Printf("pulse\tnanosecond\ttime\n")
	sc := NewTimestampFileScanner(rd)
	for sc.Scan() {
		pulseCounter++
		ns := sc.Time()
		if from.After(ns) || to.Before(ns) {
			continue
		}
		fmt.Printf("%v\t%v\t%v\n", pulseCounter, ns.UnixNano(), ns)
	}
	return sc.Err()
}
