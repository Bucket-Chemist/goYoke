package testfixture

import (
	"fmt"
	"os"
)

// PrintEnv prints the value of an environment variable.
func PrintEnv(key string) {
	fmt.Println(os.Getenv(key))
}
