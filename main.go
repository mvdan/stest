// Copyright (c) 2015, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

type record struct {
	name  string
	count int
}

type collector struct {
	buf       *bytes.Buffer
	testName  string
	anyFailed bool
	records   map[string]record
	scanner   *bufio.Scanner
}

func newCollector() *collector {
	return &collector{
		records: make(map[string]record, 0),
	}
}

func (c *collector) run(r io.Reader, w io.Writer) {
	c.scanner = bufio.NewScanner(r)
	for c.scanner.Scan() {
		line := c.scanner.Text()
		if line == "FAIL" {
			continue
		}
		if strings.HasPrefix(line, "?") || strings.HasPrefix(line, "ok") {
			// These report the overall progress, showing
			// what packages were ok or had no tests.
			c.records = make(map[string]record, 0)
			fmt.Fprintln(w, line)
			continue
		}
		if strings.HasPrefix(line, "FAIL") {
			// Some tests failed. Show the stats.
			c.finishRecord()
			for s, r := range c.records {
				fmt.Fprintf(w, "--- FAIL: %s (%d times)\n", r.name, r.count)
				i := strings.Index(s, "\n")
				if i > 0 {
					s = s[i+1:]
				}
				fmt.Fprint(w, s)
			}
			c.records = make(map[string]record, 0)
			fmt.Fprintln(w, "FAIL")
			fmt.Fprintln(w, line)
			continue
		}
		if strings.HasPrefix(line, "--- FAIL") {
			// Some test failed. Record its name and start
			// grabbing the output lines.
			c.finishRecord()
			c.testName = "Unknown"
			if sp := strings.Split(line, " "); len(sp) > 2 {
				c.testName = sp[2]
			}
			c.buf = new(bytes.Buffer)
			fmt.Fprintln(c.buf, c.testName)
			continue
		}
		if c.buf != nil {
			// Part of the test error output
			fmt.Fprintln(c.buf, line)
			continue
		}
		// We don't use these lines, so just let them
		// through. They may come from -v.
		fmt.Fprintln(w, line)
	}
}

func (c *collector) finishRecord() {
	if c.buf == nil {
		return
	}
	c.anyFailed = true
	s := c.buf.String()
	if r, e := c.records[s]; e {
		r.count++
		c.records[s] = r
	} else {
		c.records[s] = record{
			name:  c.testName,
			count: 1,
		}
	}
}

func main() {
	c := newCollector()
	c.run(os.Stdin, os.Stdout)
	if c.anyFailed {
		os.Exit(1)
	}
}
