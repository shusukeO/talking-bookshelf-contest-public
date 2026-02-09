package portfolio

import (
	"encoding/json"
	"fmt"
	"os"
)

type Portfolio struct {
	About    About        `json:"about"`
	Projects []Project    `json:"projects"`
	Skills   Skills       `json:"skills"`
	Social   []SocialLink `json:"social"`
}

type About struct {
	Name        string `json:"name"`
	Tagline     string `json:"tagline"`
	Title       string `json:"title"`
	Location    string `json:"location"`
	Education   string `json:"education"`
	CurrentWork string `json:"current_work"`
	Philosophy  string `json:"philosophy"`
}

type Project struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Users       string   `json:"users,omitempty"`
	Followers   string   `json:"followers,omitempty"`
	Years       int      `json:"years,omitempty"`
	Tech        []string `json:"tech,omitempty"`
	URL         string   `json:"url,omitempty"`
	Link        string   `json:"link,omitempty"`
	Highlight   bool     `json:"highlight,omitempty"`
}

type Skills struct {
	Backend        []string `json:"backend"`
	Frontend       []string `json:"frontend"`
	Infrastructure []string `json:"infrastructure"`
	Concepts       []string `json:"concepts"`
}

type SocialLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Link string `json:"link"`
}

func LoadPortfolio(path string) (*Portfolio, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read portfolio file: %w", err)
	}

	var portfolio Portfolio
	if err := json.Unmarshal(data, &portfolio); err != nil {
		return nil, fmt.Errorf("failed to parse portfolio JSON: %w", err)
	}

	return &portfolio, nil
}
