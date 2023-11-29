package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/feeds"
	"github.com/paulmach/orb/geojson"
)

type Bookmark struct {
	ID       string    `yaml:"id" gorm:"primary_key"`
	Title    string    `yaml:"title"`
	Rotation float32   `yaml:"rotation"`
	Extent   []float32 `yaml:"extent"`
	Content  string    `yaml:"content"`
}

type ProjectConfig struct {
	Bookmarks  map[string]map[string]Bookmark `json:"bookmarks"`
	Projection string                         `json:"projection"`
	ZoomExtent []float32                      `json:"zoom_extent"`
}

func FetchProjectConfig(config Config) (ProjectConfig, error) {
	var projectConfig ProjectConfig
	response, err := http.Get(config.ProjectURL)
	if err != nil || response.StatusCode != http.StatusOK {
		return projectConfig, fmt.Errorf("fetching project config")
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return projectConfig, fmt.Errorf("fetching project config")
	}
	err = json.Unmarshal(body, &projectConfig)
	if err != nil {
		return projectConfig, fmt.Errorf("fetching project config")
	}
	return projectConfig, nil
}

func GetQuery(query url.Values, feedConfig FeedConfig) url.Values {
	for key, value := range feedConfig.Params {
		var newValue string
		switch v := value.(type) {
		case []interface{}:
			var strValues []string
			for _, val := range v {
				strValues = append(strValues, fmt.Sprintf("%v", val))
			}
			newValue = strings.Join(strValues, ",")
		default:
			newValue = fmt.Sprintf("%v", value)
		}
		if query.Has(key) {
			query.Set(key, newValue)
		} else {
			query.Add(key, newValue)
		}
	}
	return query
}

type Options struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	UpdatedAt   string `json:"updated_at"`
}

type FeedProperties struct {
	ID          string
	Title       string
	Description string
	UpdatedAt   time.Time
}

func FromGeoJSON(original *geojson.Feature, options Options) FeedProperties {
	id := original.ID.(string)
	title := original.Properties.MustString(options.Title, "")
	description := original.Properties.MustString(options.Description, "")
	updatedAt := original.Properties.MustString(options.UpdatedAt, "")
	updatedAtTime, err := time.Parse("2006-01-02T15:04:05.999Z07:00", updatedAt)
	if err != nil {
		updatedAtTime = time.Time{}
	}

	return FeedProperties{
		id, title, description, updatedAtTime,
	}
}

/**
 * QGIS does not work well with multiple layers in a single WFS request.
 * We need to do multiple request and combine the results for multiple "TYPENAME"s.
 */
func HandleWFS(req *http.Request, config Config, feedConfig FeedConfig) (string, error) {
	requestQuery := req.URL.Query()
	extent := config.DefaultExtent
	projection := config.DefaultProjection

	options := Options{}
	jsonbody, _ := json.Marshal(feedConfig.Options)
	json.Unmarshal(jsonbody, &options)

	if requestQuery.Has("bbox") {
		_ = json.Unmarshal([]byte("["+requestQuery.Get("bbox")+"]"), &extent)
	} else if requestQuery.Has("city") {
		projectConfig, err := FetchProjectConfig(config)
		if err == nil {
			extent = projectConfig.ZoomExtent
			cityId := requestQuery.Get("city")
			if cityId == "" {
				cityId = config.Bookmarks.DefaultCity
			}
			bookmark, ok := projectConfig.Bookmarks[config.Bookmarks.Group][cityId]
			if ok {
				extent = bookmark.Extent
			}
		}
	}

	requestUrl, _ := url.Parse(config.OwsURL)
	query := requestUrl.Query()
	query.Add("VERSION", "1.1.0")
	query.Add("SERVICE", "WFS")
	query.Add("REQUEST", "GetFeature")
	query.Add("OUTPUTFORMAT", "GeoJSON")
	query.Add("STARTINDEX", "0")
	query.Add("MAXFEATURES", "20")
	query.Add("SRSNAME", projection)
	if extent[0] != 0 {
		query.Add("BBOX", fmt.Sprintf("%f,%f,%f,%f", extent[0], extent[1], extent[2], extent[3]))
	}

	allItems := []FeedProperties{}

	typeNames := []string{}
	if layerName, found := feedConfig.Params["TYPENAME"].([]interface{}); found {
		for _, layer_ := range layerName {
			typeNames = append(typeNames, layer_.(string))
		}
	} else {
		typeNames = strings.Split(feedConfig.Params["TYPENAME"].(string), ",")
	}

	for _, layer := range typeNames {
		query = GetQuery(query, feedConfig)
		query.Del("TYPENAME")
		query.Add("TYPENAME", layer)
		requestUrl.RawQuery = query.Encode()

		stringified := requestUrl.String()

		fmt.Println(stringified)

		response, err := http.Get(stringified)
		if err != nil || response.StatusCode != http.StatusOK {
			return "", err
		}

		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			return "", err
		}

		data, _ := geojson.UnmarshalFeatureCollection(body)
		err = json.Unmarshal(body, &data)
		if err != nil {
			return "", err
		}

		for _, feature := range data.Features {
			allItems = append(allItems, FromGeoJSON(feature, options))
		}
	}

	if len(typeNames) > 1 {
		sort.Slice(allItems, func(i, j int) bool {
			return allItems[j].UpdatedAt.Before(allItems[i].UpdatedAt)
		})

		maxFeatures, ok := feedConfig.Params["MAXFEATURES"].(int)
		if !ok {
			maxFeatures = 20
		}

		if len(allItems) > maxFeatures {
			allItems = allItems[:maxFeatures]
		}
	}

	feed := &feeds.Feed{
		Title:       feedConfig.Title,
		Description: feedConfig.Description,
		Link:        &feeds.Link{Href: config.BaseURL},
	}

	feed.Items = []*feeds.Item{}

	for _, feature := range allItems {
		href := fmt.Sprintf("%s/?features=%s", config.BaseURL, feature.ID)

		feed.Items = append(feed.Items, &feeds.Item{
			Id:          href,
			Title:       feature.Title,
			Description: feature.Description,
			Link:        &feeds.Link{Href: href},
			Updated:     feature.UpdatedAt,
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
