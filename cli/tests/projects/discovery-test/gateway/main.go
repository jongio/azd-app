package main

func Add(a, b int) int {
	return a + b
}

func Greet(name string) string {
	return "Hello, " + name + "!"
}

func main() {
	println(Greet("World"))
}
