package handler

import (
	"net/http"
	"sync"

	"talking-bookshelf/backend/internal/portfolio"

	"github.com/gin-gonic/gin"
)

type SocialLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type OwnerInfo struct {
	Name    string       `json:"name"`
	Tagline string       `json:"tagline"`
	Social  []SocialLink `json:"social,omitempty"`
}

var (
	ownerInfo *OwnerInfo
	ownerOnce sync.Once
)

func loadOwner() {
	ownerOnce.Do(func() {
		p, err := portfolio.LoadPortfolio("data/portfolio.json")
		if err != nil {
			return
		}
		social := make([]SocialLink, 0, len(p.Social))
		for _, s := range p.Social {
			social = append(social, SocialLink{
				Name: s.Name,
				URL:  s.URL,
			})
		}
		ownerInfo = &OwnerInfo{
			Name:    p.About.Name,
			Tagline: p.About.Tagline,
			Social:  social,
		}
	})
}

func HandleGetOwner(c *gin.Context) {
	loadOwner()
	if ownerInfo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load owner info"})
		return
	}
	c.JSON(http.StatusOK, ownerInfo)
}
