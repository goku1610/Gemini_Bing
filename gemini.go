package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

type Candidate struct {
	Index         int     `json:"Index"`
	Content       Content `json:"Content"`
	FinishReason  int     `json:"FinishReason"`
	SafetyRatings []struct {
		Category    int  `json:"Category"`
		Probability int  `json:"Probability"`
		Blocked     bool `json:"Blocked"`
	} `json:"SafetyRatings"`
	CitationMetadata interface{} `json:"CitationMetadata"`
	TokenCount       int         `json:"TokenCount"`
}

type Response struct {
	Candidates     []Candidate `json:"Candidates"`
	PromptFeedback interface{} `json:"PromptFeedback"`
	UsageMetadata  struct {
		PromptTokenCount     int `json:"PromptTokenCount"`
		CandidatesTokenCount int `json:"CandidatesTokenCount"`
		TotalTokenCount      int `json:"TotalTokenCount"`
	} `json:"UsageMetadata"`
}

type Content struct {
	Parts []string `json:"Parts"`
	Role  string   `json:"Role"`
}
type Candidates struct {
	Content *Content `json:"Content"`
}
type ContentResponse struct {
	Candidates *[]Candidates `json:"Candidates"`
}
type BingAnswer struct {
	Type            string   `json:"_type"`
	QueryContext    struct{} `json:"queryContext"`
	WebPages        WebPages `json:"webPages"`
	RelatedSearches struct{} `json:"relatedSearches"`
	RankingResponse struct{} `json:"rankingResponse"`
}

type WebPages struct {
	WebSearchURL          string   `json:"webSearchUrl"`
	TotalEstimatedMatches int      `json:"totalEstimatedMatches"`
	Value                 []Result `json:"value"`
}

type Result struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	URL              string    `json:"url"`
	IsFamilyFriendly bool      `json:"isFamilyFriendly"`
	DisplayURL       string    `json:"displayUrl"`
	Snippet          string    `json:"snippet"`
	DateLastCrawled  time.Time `json:"dateLastCrawled"`
	SearchTags       []struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	} `json:"searchTags,omitempty"`
	About []struct {
		Name string `json:"name"`
	} `json:"about,omitempty"`
}

func keywords(a string) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env key")
	}
	apiKey := os.Getenv("API_KEY")

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-pro")

	prompt := []genai.Part{
		genai.Text(a),
	}

	resp, err := model.GenerateContent(ctx, prompt...)
	if err != nil {
		log.Fatal(err)
	}

	for _, candidate := range resp.Candidates {
		fmt.Println(candidate.Content.Parts[len(candidate.Content.Parts)-1])
	}
}
func bingSearch(endpoint, token, query string) (*BingAnswer, error) {
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	param := req.URL.Query()
	param.Add("q", query)
	req.URL.RawQuery = param.Encode()

	req.Header.Add("Ocp-Apim-Subscription-Key", token)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var ans BingAnswer
	if err := json.Unmarshal(body, &ans); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	return &ans, nil
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter a sentence:")
	input, _ := reader.ReadString('\n')

	var a string = "Given a statement 'HDFC Bank statements about TATA Motors' this is the query 'site:hdfcbank.com 'tata motors'', statement = 'Amazon product reviews for Samsung Galaxy S23', query = ''samsung galaxy s23' reviews (site:amazon.in)',Statement: 'SBI car loan for Maruti Suzuki', query = 'site:sbi.co.in 'car loan maruti suzuki', Statement: 'How has the EBITDA of Bajaj Auto changed in the last one year?', query = 'after:2023 (Bajaj Auto AND EBITDA)', Statement: 'How has Motilal’s investment outlook on the oil industry changed in the last two months?', query = '[“after:2024-05-01 Gross Refining Margin“,“after:2024-05-01 Crude Prices“,“after:2024-05-01 Oil Sanctions“,“after:2024-05-01 Oil to Chemical“] ,Give the query for the statement = "
	var b string = a + input + " 'dont use 'query ='"
	file, err := os.CreateTemp("", "output.txt")

	if err != nil {
		fmt.Println("Error creating temporary file:", err)
		return
	}
	defer os.Remove(file.Name())
	old := os.Stdout
	os.Stdout = file

	keywords(b)

	os.Stdout = old

	file.Close()

	content, err := os.ReadFile(file.Name())
	if err != nil {
		fmt.Println("Error reading file content:", err)
		return
	}

	keywordsString := string(content)

	const (
		endpoint = "https://api.bing.microsoft.com/v7.0/search"
		token    = "e635fcdf348e4a868154deb206dc0740"
	)
	var searchTerm = keywordsString
	fmt.Print(searchTerm)
	ans, err := bingSearch(endpoint, token, searchTerm)
	if err != nil {
		log.Fatalf("Failed to get search results: %v", err)
	}

	var final_string string
	for _, result := range ans.WebPages.Value {
		fmt.Printf("Name: %s\nURL: %s\nDescription: %s\n\n", result.Name, result.URL, result.Snippet)
		final_string = final_string + "Name: " + result.Name + "\n" + "Description: " + result.Snippet + "\n"
	}
	final_string = final_string + "Choose the most relevant 3 to 4 " + " among these following name description pairs and give me a long summary using the final selected"
	keywords(final_string)
}
