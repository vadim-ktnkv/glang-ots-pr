package main

import (
	"fmt"

	"golang.org/x/example/hello/reverse"
)

func main() {
	// Place your code here.
	const sourceString = "Hello, OTUS!"
	resultString := reverse.String(sourceString)
	fmt.Println(resultString)
}
