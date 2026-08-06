//go:build !tinygo

package main

// The hardware entry point is built by TinyGo. This host entry point keeps the
// course and state tests independent from TinyGo's machine package.
func main() {}
