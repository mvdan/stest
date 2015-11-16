# stest

[![Build Status](https://travis-ci.org/mvdan/stest.svg?branch=master)](https://travis-ci.org/mvdan/stest)

Collect stats from `go test`.

It will deduplicate test errors and sort them by frequency. Useful to
run your tests many times and get a useful report to track down the
failing tests.

	$ go test -count 100 ./... | stest
	--- FAIL: TestFoo (8 times)
		foo_test.go:10: wanted foo, got bar
	--- FAIL: TestBar (2 times)
		foo_test.go:20: wanted bar, got foo
	FAIL
	FAIL	foo.org/bar	0.050s
