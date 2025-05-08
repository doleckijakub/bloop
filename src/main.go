package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"unicode"
	"bytes"
	"time"
	"net/url"

	"github.com/joho/godotenv"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kljensen/snowball"
	"github.com/PuerkitoBio/goquery"
)

//
// SCRAPER
//

var db *pgxpool.Pool
const MAX_SCRAPE_RECURSION_DEPTH = 4 // TODO: 10

func registerDomain(ctx context.Context, domain string) {
	var exists bool
	err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM domains WHERE domain = $1)`, domain).Scan(&exists)
	if err != nil {
		fmt.Printf("[] query failed: %v\n", err)
		return
	}
	if exists {
		return
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	supportsHttps := true

	_, err = client.Get(fmt.Sprintf("https://%s/", domain))
	if err != nil {
		supportsHttps = false
		_, err = client.Get(fmt.Sprintf("http://%s/", domain))
		if err != nil {
			fmt.Printf("[%s] failed to fetch / on http and https: %v\n", domain, err)
			return
		}
	}

	_, err = db.Exec(ctx, `INSERT INTO domains (domain, is_https) VALUES ($1, $2)`, domain, supportsHttps)
	if err != nil {
		fmt.Printf("[%s] failed to insert domain: %v\n", domain, err)
	}
}

func getDisallowedPaths(domain string) ([]string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	var resp *http.Response
	var err error

	resp, err = client.Get(fmt.Sprintf("https://%s/robots.txt", domain))
	if err != nil {
		resp, err = client.Get(fmt.Sprintf("http://%s/robots.txt", domain))
	}

	if err != nil {
		return nil, fmt.Errorf("[%s] failed to fetch robots.txt: %v", domain, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("[%s] failed to read response: %v", domain, err)
	}

	robotsTxt := string(body)
	var disallowedPaths []string

	lines := strings.Split(robotsTxt, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "User-agent:") {
			agent := strings.TrimSpace(strings.TrimPrefix(line, "User-agent:"))
			if agent == "*" {
				for j := i + 1; j < len(lines); j++ {
					nextLine := strings.TrimSpace(lines[j])
					if strings.HasPrefix(nextLine, "User-agent:") {
						break
					}
					if strings.HasPrefix(nextLine, "Disallow:") {
						path := strings.TrimSpace(strings.TrimPrefix(nextLine, "Disallow:"))
						disallowedPaths = append(disallowedPaths, path)
					}
				}
				break
			}
		}
	}

	return disallowedPaths, nil
}

func getStem(ctx context.Context, word string) (string, error) {
	var stem string
	err := db.QueryRow(ctx, `SELECT stem FROM stems WHERE word = $1`, word).Scan(&stem)
	if err == nil {
		return stem, nil
	}
	if err.Error() != "no rows in result set" {
		return "", err
	}

	stem, err = snowball.Stem(word, "english", true)
	if err == nil {
		_, insertErr := db.Exec(ctx, `INSERT INTO stems (word, stem) VALUES ($1, $2)`, word, stem)
		if insertErr != nil {
			return "", insertErr
		}
		return stem, nil
	} else {
		return "", err
	}
}

func tokenize(text string) []string {
	var words []string
	var current strings.Builder

	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else if current.Len() > 0 {
			words = append(words, current.String())
			current.Reset()
		}
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

func scrape_url(ctx context.Context, domain string, path string, disallowedPaths []string, recursion_depth int) {
	if recursion_depth > MAX_SCRAPE_RECURSION_DEPTH {
		return
	}

	var exists bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pages
			WHERE domain_id = (SELECT domain_id FROM domains WHERE domain = $1)
			AND url_path = $2
		)`, domain, path).Scan(&exists)
	if err != nil {
		fmt.Printf("[] query failed: %v\n", err)
	} else if exists {
		fmt.Printf("[%s] %s already scraped, skipping\n", domain, path)
		return
	}

	for _, disallowedPath := range disallowedPaths {
		if strings.HasPrefix(path, disallowedPath) {
			fmt.Printf("[%s] %s scraping disallowed (Disallow: %s), skipping\n", domain, path, disallowedPath)
			return
		}
	}

	var isHttps bool
	var domainID int
	err = db.QueryRow(ctx, `SELECT domain_id, is_https FROM domains WHERE domain = $1`, domain).Scan(&domainID, &isHttps)
	if err != nil {
		fmt.Printf("[] query failed: %v\n", err)
		return
	}

	protocol := "http"
	if isHttps {
		protocol = "https"
	}

	resp, err := http.Get(protocol + "://" + domain + path)
	if err != nil {
		fmt.Printf("[%s] %s failed to fetch: %v\n", domain, path, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("[%s] %s non-200 status %d for\n", domain, path, resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("[%s] %s failed to read the response: %v\n", domain, path, err)
		return
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		fmt.Printf("[%s] %s failed to parse the response: %v\n", domain, path, err)
		return
	}

	text := doc.Text()
	words := tokenize(text)

	tx, err := db.Begin(ctx)
	if err != nil {
		fmt.Printf("[%s] %s failed to begin a transaction: %v\n", domain, path, err)
		return
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `INSERT INTO pages (domain_id, url_path) VALUES ($1, $2)`, domainID, path)
	if err != nil {
		fmt.Printf("[%s] %s insert failed: %v\n", domain, path, err)
		return
	}

	var pageID int
	err = tx.QueryRow(ctx, `SELECT page_id FROM pages WHERE domain_id = $1 AND url_path = $2`, domainID, path).Scan(&pageID)
	if err != nil {
		fmt.Printf("[%s] %s query failed: %v\n", domain, path, err)
		return
	}

	termCounts := make(map[string]int)
	for _, word := range words {
		stem, err := getStem(ctx, word)
		if err == nil {
			termCounts[stem]++
		}
	}

	for term, count := range termCounts {
		var termID int
		err := tx.QueryRow(ctx, `
			INSERT INTO terms (term, document_frequency, idf)
			VALUES ($1, 1, 0.0)
			ON CONFLICT (term) DO UPDATE SET document_frequency = terms.document_frequency + 1
			RETURNING term_id`, term).Scan(&termID)
		if err != nil {
			fmt.Printf("[%s] %s insert failed: %v\n", domain, path, err)
			return
		}

		_, err = tx.Exec(ctx, `INSERT INTO page_terms (page_id, term_id, term_frequency) VALUES ($1, $2, $3)`, 
			pageID, termID, count)
		if err != nil {
			fmt.Printf("[%s] %s insert failed: %v\n", domain, path, err)
			return
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		fmt.Printf("[%s] %s commit failed: %v\n", domain, path, err)
		return
	}

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		u, err := url.Parse(href)
		if err != nil || (u.Host != "" && u.Host != domain) {
			return
		}

		relative := u.EscapedPath()
		if !strings.HasPrefix(relative, "/") {
			return
		}

		scrape_url(ctx, domain, relative, disallowedPaths, recursion_depth+1)
	})
}

func scrape_domain(domain string) {
	ctx := context.Background()

	registerDomain(ctx, domain)

	disallowedPaths, err := getDisallowedPaths(domain)
	if err != nil {
		fmt.Printf("[%s] failed to get disallowed paths from robots.txt, skipping\n", domain)
		return
	}
	
	if slices.Contains(disallowedPaths, "/") {
		fmt.Printf("[%s] scraping / disallowed, skipping\n", domain)
		return
	}

	scrape_url(ctx, domain, "/", disallowedPaths, 0)
}

func scrape_from_file(filename string) {
	file, err := os.Open(filename)
	if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

	scanner := bufio.NewScanner(file)

	// TODO: parallelize
	for scanner.Scan() {
        scrape_domain(scanner.Text())
    }

	if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }
}

//
// MAIN
//

func usage(code int) {
	fmt.Printf("Usage: %s -h,--help\n", os.Args[0])
	fmt.Printf("       %s scrape <website>\n", os.Args[0])
	fmt.Printf("       %s scrape -f <website list file>\n", os.Args[0])
	os.Exit(code)
}

func getDBConfig() (string, error) {
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	dbname := os.Getenv("POSTGRES_DB")
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")

	if user == "" || dbname == "" || host == "" || port == "" {
		return "", fmt.Errorf("missing required database configuration")
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		user,
		password,
		host,
		port,
		dbname,
	)

	return connStr, nil
}

func main() {
	// init
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	config, err := getDBConfig()
	if err != nil {
		log.Fatalf("Unable to configure database: %v\n", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, config)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v\n", err)
	}
	defer pool.Close()

	err = pool.Ping(ctx)
	if err != nil {
		log.Fatalf("Unable to ping database: %v\n", err)
	}

	// Set global db variable
	db = pool

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
			scrape_from_file(os.Args[3])
		default:
			scrape_domain(os.Args[2])
		}
	default:
		usage(1)
	}
}