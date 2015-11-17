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

type testResults struct {
	name     string
	index    int
	timesRan int
	byMsg    map[string]int

	failed  int
	percent float32
}

type collector struct {
	w         io.Writer
	buf       *bytes.Buffer
	testName  string
	anyFailed bool
	byName    map[string]testResults
}

func newCollector(w io.Writer) *collector {
	return &collector{
		w:       w,
		buf:     new(bytes.Buffer),
		byName: make(map[string]testResults, 0),
	}
}

func (c *collector) run(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
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
		name := extractTestName(line)
		if _, e := c.byName[name]; !e {
			c.byName[name] = testResults{
				name: name,
				index: len(c.byName),
				byMsg: make(map[string]int, 0),
			}
		}
		r := c.byName[name]
		r.timesRan++
		c.byName[name] = r
		c.finishRecord()
	case strings.HasPrefix(line, "?") || strings.HasPrefix(line, "ok"):
		// These report the overall progress, showing
		// what packages were ok or had no tests.
		fmt.Fprintln(c.w, line)
	case strings.HasPrefix(line, "FAIL"):
		// Package failure. Show results.
		c.finishRecord()
		for _, r := range c.sortedResults() {
			if r.percent > 0 {
				fmt.Fprintf(c.w, "--- FAIL: %s (%d times, %.2f%%)\n",
					r.name, r.failed, r.percent)
			} else {
				fmt.Fprintf(c.w, "--- FAIL: %s (%d times)\n",
					r.name, r.failed)
			}
			for msg, count := range r.byMsg {
				if len(r.byMsg) > 1 {
					fmt.Fprintf(c.w, "-- Failed %d times:\n", count)
				}
				fmt.Fprint(c.w, msg)
			}
		}
		c.byName = make(map[string]testResults, 0)
		fmt.Fprintln(c.w, "FAIL")
		fmt.Fprintln(c.w, line)
	case strings.HasPrefix(line, "--- FAIL"):
		// Single test failure.
		c.finishRecord()
		c.testName = extractTestName(line)
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
	msg := c.buf.String()
	if _, e := c.byName[c.testName]; !e {
		c.byName[c.testName] = testResults{
			name: c.testName,
			index: len(c.byName),
			byMsg: make(map[string]int, 0),
		}
	}
	r := c.byName[c.testName]
	r.failed++
	if _, e := r.byMsg[msg]; e {
		r.byMsg[msg]++
	} else {
		r.byMsg[msg] = 1
	}
	c.byName[c.testName] = r
	c.buf.Reset()
	c.testName = ""
}

type resultsList []testResults

func (rl resultsList) Len() int      { return len(rl) }
func (rl resultsList) Swap(i, j int) { rl[i], rl[j] = rl[j], rl[i] }
func (rl resultsList) Less(i, j int) bool {
	if rl[i].failed == rl[j].failed {
		return rl[i].index < rl[j].index
	}
	return rl[i].failed > rl[j].failed
}

func (c *collector) sortedResults() []testResults {
	list := make(resultsList, 0, len(c.byName))
	for n, r := range c.byName {
		r.name = n
		if r.timesRan > 0 {
			r.percent = 100*(float32(r.failed)/float32(r.timesRan))
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
