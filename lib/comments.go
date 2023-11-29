package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/aquilax/truncate"
	"github.com/gorilla/feeds"
)

type Comments struct {
	Data struct {
		Data         []Datum `json:"data"`
		CommentCount int64   `json:"commentCount"`
		PageSize     int64   `json:"pageSize"`
		PageCount    int64   `json:"pageCount"`
	} `json:"data"`
}

type Datum struct {
	ID         string    `json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	Content    string    `json:"content"`
	ByNickname string    `json:"by_nickname"`
	Page       struct {
		URL string `json:"url"`
	} `json:"page"`
	ParsedContent string `json:"parsedContent"`
}

func HandleComments(req *http.Request, config Config, feedConfig FeedConfig) (string, error) {
	requestUrl, _ := url.Parse(feedConfig.Options["url"].(string))
	query := requestUrl.Query()
	query.Add("page", "1")
	query.Add("appId", feedConfig.Options["appId"].(string))

	requestUrl.RawQuery = query.Encode()

	stringified := requestUrl.String()

	response, err := http.Get(stringified)
	if err != nil || response.StatusCode != http.StatusOK {
		return "", err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	var comments Comments
	err = json.Unmarshal(body, &comments)
	if err != nil {
		return "", err
	}

	feed := &feeds.Feed{
		Title:       feedConfig.Title,
		Description: feedConfig.Description,
		Link:        &feeds.Link{Href: config.BaseURL},
	}

	feed.Items = []*feeds.Item{}

	for _, commment := range comments.Data.Data {
		href := commment.Page.URL
		content := fmt.Sprintf("%s: %s", commment.ByNickname, truncate.Truncate(commment.Content, 50, "...", truncate.PositionEnd))

		feed.Items = append(feed.Items, &feeds.Item{
			Id:          href,
			Title:       content,
			Description: content,
			Link:        &feeds.Link{Href: href},
			Created:     commment.CreatedAt,
		})
	}

	rssFeed := (&feeds.Rss{Feed: feed}).RssFeed()
	rssFeed.Language = feedConfig.Language
	result, err := feeds.ToXML(rssFeed)
	if err != nil {
		return "", err
	}
	return result, nil
}
