// ~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~
//      /\_/\
//     ( o.o )
//      > ^ <
//
// Author: Johan Hanekom
// Date: May 2025
//
// ~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~^~

package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"

	"github.com/gocolly/colly"
)

// =============== // URL PROP STRUCT AND METHODS // ===============

type URLProps struct {
	Url       string
	Protocol  string
	Subdomain string
	Domain    string
	TLD       string
	Path      string
}

// ====> CONSTRUCTOR
func NewURLProps(url string) *URLProps {
	return &URLProps{
		Url: url,
	}
}

// ====> PARSE URL
func (u *URLProps) ParseURL(panicOnFail bool) *URLProps {
	// https://www.freecodecamp.org/news/how-to-write-a-regular-expression-for-a-url
	regexPattern := `^(https?:\/\/)?` + // protocol... maybe
		`([a-zA-Z0-9-]+\.)?` + // Subdomain... maybe
		`([a-zA-Z0-9-]+)` + // Domain
		`(\.[a-zA-Z]{2,}(?:\.[a-zA-Z]{2,})*)` + // TLD (supports .co.za, .ac.za, etc.)
		`(\/[a-zA-Z0-9\/._-]*)?$` // Path ... maybe

	re := regexp.MustCompile(regexPattern)
	match := re.FindStringSubmatch(u.Url)

	if match == nil || len(match) < 6 {
		if panicOnFail {
			log.Fatal("Failed to parse URL:", u.Url)
		}
		return u
	}

	u.Protocol = match[1]
	u.Subdomain = match[2]
	u.Domain = match[3]
	u.TLD = match[4]
	u.Path = match[5]
	if u.Protocol == "" {
		u.Protocol = "http"
	}
	return u
}

// ====> PRINT TO CONSOLE
func (u *URLProps) Print() {
	fmt.Printf("Parsed URL: [Protocol: %s | Subdomain: %s | Domain: %s | TLD: %s | Path: %s]\n",
		u.Protocol, u.Subdomain, u.Domain, u.TLD, u.Path)
}

// ====> SLICE OF PROPS (MAIN DATA STRUCTURE)
var urlMatches []URLProps

// =============== // DATA SAVING FUNCTIONS // ===============

func WriteCSV(urlMatches []URLProps, target URLProps) error {
	// ====> FILE OPEN
	csvFile, csvErr := os.Create("site-map-" + target.Domain + ".csv")
	if csvErr != nil {
		return fmt.Errorf("failed to create the output CSV file: %w", csvErr)
	}
	defer csvFile.Close()

	// ====> CREATE A WRITER
	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	// ====> WRITE HEADERS
	headers := []string{
		"Url",
		"Protocol",
		"Subdomain",
		"Domain",
		"TLD",
		"Path",
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}

	// ====> CREATE A STRING SLICE (PER ROW)
	for _, urlmatch := range urlMatches {
		record := []string{
			urlmatch.Url,
			urlmatch.Protocol,
			urlmatch.Subdomain,
			urlmatch.Domain,
			urlmatch.TLD,
			urlmatch.Path,
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}
	return nil
}

func WriteTXT(urlMatches []URLProps, target URLProps) error {
	txtFile, txtErr := os.Create("site-map-" + target.Domain + ".txt")
	if txtErr != nil {
		return fmt.Errorf("failed to create the output TXT file: %w", txtErr)
	}
	defer txtFile.Close()

	for _, urlmatch := range urlMatches {
		if _, err := fmt.Fprintln(txtFile, urlmatch.Url); err != nil {
			return fmt.Errorf("failed to write TXT record: %w", err)
		}
	}
	return nil
}

func WriteJSON(urlMatches []URLProps, target URLProps) error {
	jsonFile, jsonErr := os.Create("site-map-" + target.Domain + ".json")
	if jsonErr != nil {
		return fmt.Errorf("failed to create the output JSON file: %w", jsonErr)
	}
	defer jsonFile.Close()

	encoder := json.NewEncoder(jsonFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(urlMatches); err != nil {
		return fmt.Errorf("failed to write JSON data: %w", err)
	}
	return nil
}

func main() {
	// ====> PARSE THE FLAG
	fmt.Println("CRALWER...")
	visited := make(map[string]struct{})

	targetUrl := flag.String("u", "https://brightdata.com", "URL to scrape")
	fileType := flag.String("f", "a", "File type")
	flag.Parse()

	// ====> CONSTRUCT, PARSE AND PRINT
	target := NewURLProps(*targetUrl)
	target.ParseURL(true)
	target.Print()

	// ====> CREATE A COLLECTOR AND ONLY ALLOW VISITS TO THE SPECIFIED DOMAIN
	c := colly.NewCollector()
	c.UserAgent =
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
			"AppleWebKit/537.36 (KHTML, like Gecko) " +
			"Chrome/111.0.0.0 Safari/537.36"

	// =============== // EVENT LISTENERS // ===============
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		// ====> GET THE ABSOLUTE LINK
		link := e.Request.AbsoluteURL(e.Attr("href"))
		if link == "" {
			return
		}

		// ====> ONLY CONSIDER THE LINK IF IT HASN'T BEEN SEEN BEFORE
		if _, found := visited[link]; !found {
			// ====> CREATE A NEW URL PROP AND PARSE IT
			result := NewURLProps(link)
			result.ParseURL(false)

			// ====> SAVE TO MAP
			visited[link] = struct{}{}

			// ====> ONLY SAVE IF I HAVE ENOUGHT VALID DATA
			if result.Domain != "" && result.TLD != "" && result.Domain == target.Domain && target.Subdomain != "languagecentre" {
				urlMatches = append(urlMatches, *result)
				e.Request.Visit(link)
			}

		}
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	// c.OnScraped(func(r *colly.Response) {
	// 	fmt.Println("Finished", r.Request.URL)
	// })

	// =============== // ENTRY POINT // ===============

	c.Visit(target.Url)

	// =============== // WRITE OUTPUTS TO FILE // ===============

	sort.Slice(urlMatches, func(i, j int) bool {
		return urlMatches[i].Url < urlMatches[j].Url
	})

	switch *fileType {
	case "a":
		fmt.Println("Writing CSV, JSON and TXT files...")
		if err := WriteCSV(urlMatches, *target); err != nil {
			log.Fatalln("Failed to write CSV file", err)
		}

		if err := WriteJSON(urlMatches, *target); err != nil {
			log.Fatalln("Failed to write JSON file", err)
		}

		if err := WriteTXT(urlMatches, *target); err != nil {
			log.Fatalln("Failed to write TXT file", err)
		}
	case "c":
		fmt.Println("Writing CSV file...")
		if err := WriteCSV(urlMatches, *target); err != nil {
			log.Fatalln("Failed to write CSV file", err)
		}
	case "j":
		fmt.Println("Writing JSON file...")
		if err := WriteJSON(urlMatches, *target); err != nil {
			log.Fatalln("Failed to write JSON file", err)
		}
	case "t":
		if err := WriteTXT(urlMatches, *target); err != nil {
			log.Fatalln("Failed to write TXT file", err)
		}
	default:
		fmt.Println("Not a valid file type. Not writing....")
	}

	fmt.Println("Scraping completed")
}
