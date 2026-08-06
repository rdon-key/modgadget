//go:build !tinygo

package main

// The hardware example is built by TinyGo. This no-op entry point keeps host
// package checks independent from TinyGo's machine package.
func main() {}
