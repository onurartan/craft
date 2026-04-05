package main

import "fmt"

var AppVersion = "dev" // Overridden by 
var AppVersion_Date = "date"

func HelloCraft() {
	fmt.Println("Hello, Craft!")
	fmt.Println("Welcome to the future of Go development with hot-reload capabilities.")
	fmt.Println("This is a simple example to demonstrate the structure of the Craft engine.")
	fmt.Println("Please refer to the main.go file for the entry point of the application.")
}

func main() {
	HelloCraft()
	fmt.Println("Version:", AppVersion)
	fmt.Println("Version Date", AppVersion_Date)
}
