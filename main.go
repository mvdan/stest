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
	w         io.Writer
	buf       *bytes.Buffer
	testName  string
	anyFailed bool
	records   map[string]record
	scanner   *bufio.Scanner
	curIndex  int
}

func newCollector(w io.Writer) *collector {
	return &collector{
		w:       w,
		buf:     new(bytes.Buffer),
		records: make(map[string]record, 0),
	}
}

func (c *collector) run(r io.Reader) {
	c.scanner = bufio.NewScanner(r)
	for c.scanner.Scan() {
		line := c.scanner.Text()
		c.parseLine(line)
	}
}

func (c *collector) parseLine(line string) {
	switch {
	case line == "FAIL" || line == "PASS":
	case strings.HasPrefix(line, "exit status"):
	case strings.HasPrefix(line, "=== RUN"):
		c.finishRecord()
	case strings.HasPrefix(line, "?") || strings.HasPrefix(line, "ok"):
		// These report the overall progress, showing
		// what packages were ok or had no tests.
		fmt.Fprintln(c.w, line)
	case strings.HasPrefix(line, "FAIL"):
		// Package failure. Show results.
		c.finishRecord()
		for _, r := range c.sortedRecords() {
			fmt.Fprintf(c.w, "--- FAIL: %s (%d times)\n", r.name, r.count)
			fmt.Fprint(c.w, r.msg)
		}
		c.records = make(map[string]record, 0)
		fmt.Fprintln(c.w, "FAIL")
		fmt.Fprintln(c.w, line)
	case strings.HasPrefix(line, "--- FAIL"):
		// Single test failure.
		c.finishRecord()
		c.testName = "Unknown"
		if sp := strings.Split(line, " "); len(sp) > 2 {
			c.testName = sp[2]
		}
		fmt.Fprintln(c.buf, c.testName)
	case c.testName != "":
		// Part of the current test error output
		fmt.Fprintln(c.buf, line)
	}
}

func (c *collector) finishRecord() {
	if c.testName == "" {
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
	c.buf.Reset()
	c.testName = ""
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

func (c *collector) sortedRecords() []record {
	list := make(recordList, 0, len(c.records))
	for s, r := range c.records {
		if i := strings.Index(s, "\n"); i > 0 {
			s = s[i+1:]
		}
		r.msg = s
		list = append(list, r)
	}
	sort.Sort(list)
	return list
}

func main() {
	c := newCollector(os.Stdout)
	c.run(os.Stdin)
	if c.anyFailed {
		os.Exit(1)
	}
}
