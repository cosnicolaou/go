// Command mbox-egrep applies a regular expression and template (as
// per go's regexp.Expand method) to each of the messages in the specified
// mbox file.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/mail"
	"os"
	"regexp"
	"strings"

	mbox "github.com/galdor/go-mbox"
)

var (
	patternFlag, templateFlag string
)

func init() {
	flag.StringVar(&patternFlag, "pattern", "", "regular expression for regexp.Expand")
	flag.StringVar(&templateFlag, "template", "", "template for regexp.Expand")
}

func exitf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, strings.TrimSuffix(format, "\n")+"\n", args...)
	os.Exit(1)
}

func handleParts(contentHeader string, body []byte) ([][]byte, error) {
	mediaType, params, err := mime.ParseMediaType(contentHeader)
	if err != nil {
		return nil, err
	}
	bodies := make([][]byte, 0, 10)
	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		mr := multipart.NewReader(bytes.NewReader(body), boundary)
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("failed reading part header: %v", err)
			}
			if ch := p.Header.Get("Content-Type"); strings.HasPrefix(ch, "multipart") {
				sub, err := handleParts(ch, body)
				if err != nil {
					return nil, fmt.Errorf("failed parsed nested parts: %v", err)
				}
				bodies = append(bodies, sub...)
			} else {
				body, err := ioutil.ReadAll(p)
				if err != nil {
					return nil, fmt.Errorf("failed reading  part body: %v", err)
				}
				bodies = append(bodies, body)
			}
			p.Close()
		}
	}
	return bodies, nil
}

func main() {
	flag.Parse()

	pattern := regexp.MustCompile(patternFlag)
	template := []byte(templateFlag)

	for _, path := range flag.Args() {
		mbox, err := mbox.Open(path, mbox.Mboxrd)
		if err != nil {
			exitf("failed to open: %v: %v", path, err)
		}

		for {
			msg, err := mbox.Read()
			if err != nil {
				exitf("failed to read from %v: %v", path, err)
			}
			if msg == nil || msg.Data == nil {
				break
			}

			mrd := bytes.NewReader(msg.Data)
			emsg, err := mail.ReadMessage(mrd)
			if err != nil {
				exitf("failed to parse message: %v: %v", msg.Id, err)
			}
			body, err := ioutil.ReadAll(emsg.Body)
			if err != nil {
				exitf("failed to read body: %v: %v", msg.Id, err)
			}
			parts, err := handleParts(emsg.Header.Get("Content-Type"), body)
			if err != nil {
				exitf("failed reading parts: %v: %v\n", msg.Id, err)
			}
			dst := make([]byte, 0, 16*1024)
			for _, part := range parts {
				for _, submatches := range pattern.FindAllSubmatchIndex(part, -1) {
					tmp := pattern.Expand(dst, template, part, submatches)
					if len(tmp) == 0 {
						continue
					}
					dst = append(dst, tmp...)
					dst = append(dst, '\n')
				}
			}
			if len(dst) == 0 {
				continue
			}
			if _, err := os.Stdout.Write(dst); err != nil {
				exitf("failed to write to stdout: %v", err)
			}
		}
		mbox.Close()
	}
}
