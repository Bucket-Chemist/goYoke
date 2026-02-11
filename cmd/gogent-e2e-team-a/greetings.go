package main

import "fmt"

func main() {
	// Create three greetings in different languages
	english := NewGreeting("English", "Hello")
	spanish := NewGreeting("Spanish", "Hola")
	japanese := NewGreeting("Japanese", "Konnichiwa")

	// Print each greeting using String() method via fmt.Println
	fmt.Println(english)
	fmt.Println(spanish)
	fmt.Println(japanese)
}
