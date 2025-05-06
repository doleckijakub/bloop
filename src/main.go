package main

import (
	_ "embed"
	"strings"
	"fmt"
	"os"
	"net/http"
)

//go:embed data/million.txt
var million []byte

func scrape() {
	fmt.Println("scraping...\n")
	for _, domain := range strings.Split(strings.TrimRight(string(million), "\n"), "\n") {
		fmt.Println(domain)
	}
}

func root_handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "req: %s\n", r.URL.Path)
}

func serve(port string) {
	address := ":" + port

	http.HandleFunc("/", root_handler)
	fmt.Printf("Starting server on %s\n", address)
	err := http.ListenAndServe(address, nil)
	if err != nil {
		fmt.Println("Error starting server: ", err)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: %s scrape")
		fmt.Println("       %s serve [port]")
		return
	}

	switch os.Args[1] {
	case "scrape":
		scrape()
	case "serve":
		port := "8080"

		if len(os.Args) == 3 {
			port = os.Args[2]
		}

		serve(port)
	}
}
