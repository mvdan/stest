// Copyright (c) 2015, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

type record struct {
	name  string
	count int
	msg   string
	index int
}

type collector struct {
	buf       *bytes.Buffer
	testName  string
	anyFailed bool
	records   map[string]record
	scanner   *bufio.Scanner
	curIndex  int
}

type recordList []record

func (rl recordList) Len() int      { return len(rl) }
func (rl recordList) Swap(i, j int) { rl[i], rl[j] = rl[j], rl[i] }
func (rl recordList) Less(i, j int) bool {
	if rl[i].count == rl[j].count {
		return rl[i].index < rl[j].index
	}
	return rl[i].count > rl[j].count
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
		switch {
		case line == "FAIL" || line == "PASS":
		case strings.HasPrefix(line, "exit status"):
		case strings.HasPrefix(line, "?") || strings.HasPrefix(line, "ok"):
			// These report the overall progress, showing
			// what packages were ok or had no tests.
			fmt.Fprintln(w, line)
		case strings.HasPrefix(line, "FAIL"):
			// Some tests failed. Show the stats.
			c.finishRecord()
			list := make(recordList, 0, len(c.records))
			for s, r := range c.records {
				if i := strings.Index(s, "\n"); i > 0 {
					s = s[i+1:]
				}
				r.msg = s
				list = append(list, r)
			}
			sort.Sort(list)
			for _, r := range list {
				fmt.Fprintf(w, "--- FAIL: %s (%d times)\n", r.name, r.count)
				fmt.Fprint(w, r.msg)
			}
			c.records = make(map[string]record, 0)
			fmt.Fprintln(w, "FAIL")
			fmt.Fprintln(w, line)
		case strings.HasPrefix(line, "--- FAIL"):
			// Some test failed. Record its name and start
			// grabbing the output lines.
			c.finishRecord()
			c.testName = "Unknown"
			if sp := strings.Split(line, " "); len(sp) > 2 {
				c.testName = sp[2]
			}
			c.buf = new(bytes.Buffer)
			fmt.Fprintln(c.buf, c.testName)
		case c.buf != nil:
			// Part of the test error output
			fmt.Fprintln(c.buf, line)
		}
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
			index: c.curIndex,
		}
		c.curIndex++
	}
}

func main() {
	c := newCollector()
	c.run(os.Stdin, os.Stdout)
	if c.anyFailed {
		os.Exit(1)
	}
}
