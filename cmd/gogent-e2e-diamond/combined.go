package main

import "fmt"

func main() {
	p := NewPair("hello", "world")
	fmt.Printf("Pair: Left=%s, Right=%s\n", p.Left, p.Right)
	fmt.Printf("Reversed Left: %s\n", Reverse(p.Left))
}
