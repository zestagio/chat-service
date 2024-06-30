package main

import (
	"log"
	"net/http"
)

const addr = ":3001"

func main() {
	log.Printf("Look at http://localhost%v/", addr)
	if err := http.ListenAndServe(addr, http.FileServer(http.Dir("static"))); err != nil { //nolint:gosec // non-prod solution
		log.Fatal(err)
	}
}
