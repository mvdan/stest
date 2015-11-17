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
	out      io.Writer
	buf      *bytes.Buffer
	testName string
	byName   map[string]testResults
}

func newCollector(w io.Writer) *collector {
	return &collector{
		out:    w,
		buf:    new(bytes.Buffer),
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

func (c *collector) getTestResults(name string) testResults {
	if _, e := c.byName[name]; !e {
		c.byName[name] = testResults{
			name:  name,
			index: len(c.byName),
			byMsg: make(map[string]int, 0),
		}
	}
	return c.byName[name]
}

func (c *collector) printResults() {
	for _, r := range c.sortedResults() {
		if r.percent > 0 {
			fmt.Fprintf(c.out, "--- FAIL: %s (%d times, %.2f%%)\n",
				r.name, r.failed, r.percent)
		} else {
			fmt.Fprintf(c.out, "--- FAIL: %s (%d times)\n",
				r.name, r.failed)
		}
		for msg, count := range r.byMsg {
			if len(r.byMsg) > 1 {
				fmt.Fprintf(c.out, "-- Failed %d times:\n", count)
			}
			fmt.Fprint(c.out, msg)
		}
	}
}

func (c *collector) parseLine(line string) {
	switch {
	case line == "FAIL" || line == "PASS":
	case strings.HasPrefix(line, "exit status"):

	case strings.HasPrefix(line, "=== RUN"):
		name := extractTestName(line)
		r := c.getTestResults(name)
		r.timesRan++
		c.byName[name] = r
		c.finishRecord()

	case strings.HasPrefix(line, "?") || strings.HasPrefix(line, "ok"):
		// These report the overall progress, showing
		// what packages were ok or had no tests.
		fmt.Fprintln(c.out, line)

	case strings.HasPrefix(line, "FAIL"):
		// Package failure. Show results.
		c.finishRecord()
		c.printResults()
		c.byName = make(map[string]testResults, 0)
		fmt.Fprintln(c.out, "FAIL")
		fmt.Fprintln(c.out, line)

	case strings.HasPrefix(line, "--- FAIL"):
		// Single test failure.
		c.finishRecord()
		if c.testName == "" {
			c.testName = extractTestName(line)
		}

	case c.testName != "":
		// Part of the current test error output
		fmt.Fprintln(c.buf, line)
	}
}

func (c *collector) finishRecord() {
	if c.testName == "" {
		return
	}
	msg := c.buf.String()
	r := c.getTestResults(c.testName)
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
			r.percent = 100 * (float32(r.failed) / float32(r.timesRan))
		}
		list = append(list, r)
	}
	sort.Sort(list)
	return list
}

func main() {
	c := newCollector(os.Stdout)
	c.run(os.Stdin)
	if len(c.byName) > 0 {
		os.Exit(1)
	}
}
