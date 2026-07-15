package main

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: hash-password <password>")
		os.Exit(2)
	}
	h, e := bcrypt.GenerateFromPassword([]byte(os.Args[1]), bcrypt.DefaultCost)
	if e != nil {
		panic(e)
	}
	fmt.Println(string(h))
}
