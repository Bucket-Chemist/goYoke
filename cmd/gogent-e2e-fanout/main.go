package main

import "fmt"

func main() {
	red := NewColor("red", "FF0000")
	blue := NewColor("blue", "0000FF")

	fmt.Println(red)
	fmt.Println(blue)
}
