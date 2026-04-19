package pkg

// Hello returns a greeting string.
func Hello() string {
	return helper()
}

func helper() string {
	return "hello from pkg"
}
