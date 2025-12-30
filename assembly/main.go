package main

import (
	"fmt"
	"strings"
	"time"
)

func buildGreeting() string {
	words := []string{"Hello", "World"}
	wave := "~" + strings.Repeat("*", 5) + "~"
	return fmt.Sprintf("%s %s %s", words[0], wave, words[1])
}

func main() {
	const emoji = "(^_^)/"
	const tagline = "Greetings from Go with a splash of sunshine!"

	fmt.Println(strings.Repeat("=", len(tagline)))
	fmt.Println(tagline)
	fmt.Println(strings.Repeat("=", len(tagline)))

	fmt.Printf("%s %s\n", emoji, buildGreeting())

	now := time.Now().Format("Mon 02 Jan 2006 15:04:05 MST")
	fmt.Printf("Time stamp: %s\n", now)
}
