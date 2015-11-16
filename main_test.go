// Copyright (c) 2015, Daniel Martí <mvdan@mvdan.cc>
// See LICENSE for licensing information

package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"path/filepath"
	"testing"
)

var write = flag.Bool("w", false, "Write test output files")

func init() {
	flag.Parse()
}

func getOut(in string) []byte {
	r := bytes.NewBufferString(in)
	w := new(bytes.Buffer)
	c := newCollector()
	c.run(r, w)
	return w.Bytes()
}

func doTest(t *testing.T, testName, in, exp string) {
	got := string(getOut(in))
	if got != exp {
		t.Errorf("Unexpected output in test: %s\nExpected:\n%s\nGot:\n%s\n",
			testName, exp, got)
	}
}

func TestCases(t *testing.T) {
	inPaths, err := filepath.Glob(filepath.Join("testdata", "*.in.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, inPath := range inPaths {
		base := inPath[:len(inPath)-len(".in.txt")]
		outPath := base + ".out.txt"
		testName := base[len("testdata")+1:]
		in, err := ioutil.ReadFile(inPath)
		if err != nil {
			t.Fatal(err)
		}
		if *write {
			out := getOut(string(in))
			if err := ioutil.WriteFile(outPath, out, 0644); err != nil {
				t.Fatal(err)
			}
		} else {
			exp, err := ioutil.ReadFile(outPath)
			if err != nil {
				t.Fatal(err)
			}
			doTest(t, testName, string(in), string(exp))
		}
	}
}
