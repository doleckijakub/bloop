package main

import (
	// "context"
	"fmt"
	// "io"
	"log"
	// "net/http"
	"os"
	// "strings"
	// "time"

	"github.com/joho/godotenv"
	// "github.com/jackc/pgx/v5/pgxpool"
)

//
// SCRAPER
//

func can_scrape_domain(domain *string) {
	// urls := []string{
	// 	fmt.Sprintf("https://%s/robots.txt", domain),
	// 	fmt.Sprintf("http://%s/robots.txt", domain),
	// }

	// var resp *http.Response
	// var err error
	// var finalURL string

	// for _, url := range urls {
	// 	resp, err = client.Get(url)
	// 	if err == nil {
	// 		finalURL = url
	// 		break
	// 	}
	// }

	// if err != nil {
	// 	fmt.Printf("!!! [%s] Failed to fetch robots.txt: %v\n", domain, err)
	// 	continue
	// }
	// defer resp.Body.Close()

	// body, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	fmt.Printf("!!! [%s] Failed to read response: %v\n", domain, err)
	// 	continue
	// }

	// robotsTxt := string(body)
	// scrapingAllowed := true

	// lines := strings.Split(robotsTxt, "\n")
	// for i := 0; i < len(lines); i++ {
	// 	line := strings.TrimSpace(lines[i])
	// 	if strings.HasPrefix(line, "User-agent:") {
	// 		agent := strings.TrimSpace(strings.TrimPrefix(line, "User-agent:"))
	// 		if agent == "*" {
	// 			for j := i + 1; j < len(lines); j++ {
	// 				nextLine := strings.TrimSpace(lines[j])
	// 				if strings.HasPrefix(nextLine, "User-agent:") {
	// 					break
	// 				}
	// 				if strings.HasPrefix(nextLine, "Disallow:") {
	// 					path := strings.TrimSpace(strings.TrimPrefix(nextLine, "Disallow:"))
	// 					if path == "/" {
	// 						scrapingAllowed = false
	// 						break
	// 					}
	// 				}
	// 			}
	// 			break
	// 		}
	// 	}
	// }

	// return scrapingAllowed
}

func scrape_domain(domain *string) {
	fmt.Printf("scraping %s\n", *domain)
}

func scrape_from_file(filename *string) {
	fmt.Printf("scraping from %s\n", *filename)

	// client := &http.Client{
	// 	Timeout: 5 * time.Second,
	// }

	// for _, domain := range strings.Split(strings.TrimRight(string(million), "\n"), "\n") {
	// 	if domain == "" {
	// 		continue
	// 	}

	// 	scrape_domain(&domain)
	// }
}

//
// SERVER
//

// func root_handler(w http.ResponseWriter, r *http.Request) {
// 	fmt.Fprintf(w, "req: %s\n", r.URL.Path)
// }

// func serve(port string) {
// 	address := ":" + port

// 	http.HandleFunc("/", root_handler)
// 	fmt.Printf("Starting server on %s\n", address)
// 	err := http.ListenAndServe(address, nil)
// 	if err != nil {
// 		fmt.Println("Error starting server: ", err)
// 	}
// }

//
// MAIN
//

func usage(code int) {
	fmt.Printf("Usage: %s -h,--help\n", os.Args[0])
	fmt.Printf("       %s scrape <website>\n", os.Args[0])
	fmt.Printf("       %s scrape -f <website list file>\n", os.Args[0])
	// fmt.Printf("       %s serve [port]\n", os.Args[0])
	os.Exit(code)
}

func main() {
	// init

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	// parse args

	if len(os.Args) < 2 {
		usage(1)
	}

	switch os.Args[1] {
	case "-h", "--help":
		usage(0)
	case "scrape":
		if len(os.Args) < 3 {
			usage(1)
		}

		switch os.Args[2] {
		case "-f":
			if len(os.Args) < 4 {
				usage(1)
			}

			scrape_from_file(&os.Args[3])
		default:
			scrape_domain(&os.Args[2])
		}
	// case "serve":
	// 	port := "8080"

	// 	if len(os.Args) < 3 {
	// 		port = os.Args[2]
	// 	}

	// 	serve(port)
	}
}
