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
	name    string
	index   int
	msg     string
	failed  int
	percent float32
}

type collector struct {
	w         io.Writer
	buf       *bytes.Buffer
	testName  string
	anyFailed bool
	records   map[string]record
	timesRan  map[string]int
	scanner   *bufio.Scanner
	curIndex  int
}

func newCollector(w io.Writer) *collector {
	return &collector{
		w:       w,
		buf:     new(bytes.Buffer),
		records: make(map[string]record, 0),
		timesRan: make(map[string]int, 0),
	}
}

func (c *collector) run(r io.Reader) {
	c.scanner = bufio.NewScanner(r)
	for c.scanner.Scan() {
		line := c.scanner.Text()
		c.parseLine(line)
	}
}

func extractTestName(line string) string {
	if fields := strings.Fields(line); len(fields) > 2 {
		return fields[2]
	}
	return "Unknown"
}

func (c *collector) parseLine(line string) {
	switch {
	case line == "FAIL" || line == "PASS":
	case strings.HasPrefix(line, "exit status"):
	case strings.HasPrefix(line, "=== RUN"):
		testName := extractTestName(line)
		if _, e := c.timesRan[testName]; e {
			c.timesRan[testName]++
		} else {
			c.timesRan[testName] = 1
		}
		c.finishRecord()
	case strings.HasPrefix(line, "?") || strings.HasPrefix(line, "ok"):
		// These report the overall progress, showing
		// what packages were ok or had no tests.
		fmt.Fprintln(c.w, line)
	case strings.HasPrefix(line, "FAIL"):
		// Package failure. Show results.
		c.finishRecord()
		for _, r := range c.sortedRecords() {
			if r.percent > 0 {
				fmt.Fprintf(c.w, "--- FAIL: %s (%d times, %.2f%%)\n",
					r.name, r.failed, r.percent)
			} else {
				fmt.Fprintf(c.w, "--- FAIL: %s (%d times)\n",
					r.name, r.failed)
			}
			fmt.Fprint(c.w, r.msg)
		}
		c.records = make(map[string]record, 0)
		c.timesRan = make(map[string]int, 0)
		fmt.Fprintln(c.w, "FAIL")
		fmt.Fprintln(c.w, line)
	case strings.HasPrefix(line, "--- FAIL"):
		// Single test failure.
		c.finishRecord()
		c.testName = extractTestName(line)
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
		r.failed++
		c.records[s] = r
	} else {
		c.records[s] = record{
			name:   c.testName,
			failed: 1,
			index:  c.curIndex,
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
	if rl[i].failed == rl[j].failed {
		return rl[i].index < rl[j].index
	}
	return rl[i].failed > rl[j].failed
}

func (c *collector) sortedRecords() []record {
	list := make(recordList, 0, len(c.records))
	for s, r := range c.records {
		if i := strings.Index(s, "\n"); i > 0 {
			s = s[i+1:]
		}
		r.msg = s
		ran := c.timesRan[r.name]
		if ran > 0 {
			r.percent = 100*(float32(r.failed)/float32(ran))
		}
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
