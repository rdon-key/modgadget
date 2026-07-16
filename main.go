

package main

import "time"

func main() {
	println("Hello from M5Stack Stamp-S3A!")

	for {
		println("alive")
		time.Sleep(time.Second)
	}
}
