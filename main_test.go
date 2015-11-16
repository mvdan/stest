// Copyright (c) 2015, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

var (
	write = flag.Bool("w", false, "Write test output files")

	testNameRegexp = regexp.MustCompile(`testdata/(.*)\.in\.txt`)
)

func init() {
	flag.Parse()
}

func getOut(in io.Reader) []byte {
	w := new(bytes.Buffer)
	c := newCollector(w)
	c.run(in)
	return w.Bytes()
}

func doTest(t *testing.T, testName string, in io.Reader, exp string) {
	got := string(getOut(in))
	if got != exp {
		t.Errorf("Unexpected output in test: %s\nExpected:\n%s\nGot:\n%s\n",
			testName, exp, got)
	}
}

func testCase(t *testing.T, testName string) {
	inPath := filepath.Join("testdata", testName+".in.txt")
	outPath := filepath.Join("testdata", testName+".out.txt")
	in, err := os.Open(inPath)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	if *write {
		out := getOut(in)
		if err := ioutil.WriteFile(outPath, out, 0644); err != nil {
			t.Fatal(err)
		}
	} else {
		exp, err := ioutil.ReadFile(outPath)
		if err != nil {
			t.Fatal(err)
		}
		doTest(t, testName, in, string(exp))
	}
}

func TestCases(t *testing.T) {
	inPaths, err := filepath.Glob(filepath.Join("testdata", "*.in.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, inPath := range inPaths {
		m := testNameRegexp.FindStringSubmatch(inPath)
		if m == nil {
			continue
		}
		testName := m[1]
		testCase(t, testName)
	}
}
