package services

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

const MAX_POST_SIZE = 25 * 1024 * 1024

type E621Post struct {
	ID   int    `json:"id"`
	URL  string `json:"url"`
	Ext  string `json:"file_ext"`
	Size int    `json:"file_size"`
}

type E621PostResponse struct {
	ID   int `json:"id"`
	File struct {
		URL  string `json:"url"`
		Ext  string `json:"ext"`
		Size int    `json:"size"`
	} `json:"file"`
	Sample struct {
		URL  string `json:"url"`
		Has  bool   `json:"has"`
		Alts map[string]struct {
			URLs []*string `json:"urls"`
		} `json:"alternates"`
	} `json:"sample"`
}

type IE621Service interface {
	GetRandomPost() (*E621Post, error)
	GetPostByID(id int) (*E621Post, error)
	SearchPosts(tags string, limit, page int) ([]*E621Post, error)
	GetPopularPosts() ([]*E621Post, error)
}

type E621Service struct {
	httpClient *http.Client
	userAgent  string
	logger     zerolog.Logger
}

func NewE621Service(userAgent string, parent zerolog.Logger) *E621Service {
	return &E621Service{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
		userAgent: userAgent,
		logger:    parent.With().Str("service", "e621").Logger(),
	}
}

func (e *E621Service) GetRandomPost() (*E621Post, error) {
	const url = "https://e621.net/posts/random.json"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", e.userAgent)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var post struct {
		Post E621PostResponse `json:"post"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
		return nil, err
	}

	return e.ParsePost(&post.Post)
}

func (e *E621Service) GetPostByID(id int) (*E621Post, error) {
	url := fmt.Sprintf("https://e621.net/posts/%d.json", id)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", e.userAgent)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var post struct {
		Post E621PostResponse `json:"post"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
		return nil, err
	}

	return e.ParsePost(&post.Post)
}

func (e *E621Service) SearchPosts(tags string, limit, page int) ([]*E621Post, error) {
	// URL encode the query
	tags = url.QueryEscape(tags)

	// Append the limit and page
	url := "https://e621.net/posts.json?tags=%s&limit=%d&page=%d"
	url = fmt.Sprintf(url, tags, limit, page)

	e.logger.Debug().Msgf("Search URL: %s", url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", e.userAgent)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var posts struct {
		Posts []*E621PostResponse `json:"posts"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		return nil, err
	}

	var result []*E621Post
	for _, post := range posts.Posts {
		parsed, err := e.ParsePost(post)
		if err != nil {
			continue
		}

		result = append(result, parsed)
	}

	return result, nil
}

type File struct {
	ContentLength int
	URL           string
}

func (e *E621Service) GetContentLength(url string) (int, error) {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("User-Agent", e.userAgent)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.ContentLength != -1 {
		return int(resp.ContentLength), nil
	}

	// If the content length is not provided, we have to download the file
	resp, err = e.httpClient.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return int(resp.ContentLength), nil
}

func (e *E621Service) FindSuitableSample(post *E621PostResponse) (string, error) {
	useSample := post.File.Size > MAX_POST_SIZE
	isVideo := post.File.Ext == "webm" || post.File.Ext == "mp4"

	// If the post has no file URL, return an error
	if post.File.URL == "" {
		return "", errors.New("post is hidden for bots (requires login)")
	}

	// If sample is not required, just return the original
	if !useSample {
		return post.File.URL, nil
	}

	// If the post is too large and we can use the sample, use it
	if !post.Sample.Has {
		return "", errors.New("post is too large and has no samples")
	}

	// If the post is a video, we find the best fit
	if isVideo {
		possible := []File{}

		for _, alt := range post.Sample.Alts {
			var file *File = nil

			for _, url := range alt.URLs {
				// Skip if the URL is nil
				if url == nil || *url == "" {
					continue
				}

				// Get the content length
				length, err := e.GetContentLength(*url)
				if err != nil {
					continue
				}

				// Skip if the file is too large
				if length > MAX_POST_SIZE {
					continue
				}

				// If the file is smaller than the current file, replace it
				if file == nil || length < file.ContentLength {
					file = &File{
						ContentLength: length,
						URL:           *url,
					}
				}
			}

			if file != nil {
				possible = append(possible, *file)
			}
		}

		// If we have no possible files, return an error
		if len(possible) == 0 {
			return "", errors.New("no suitable samples found")
		}

		// Find the biggest file
		var biggest *File
		for _, file := range possible {
			if biggest == nil || file.ContentLength > biggest.ContentLength {
				biggest = &file
			}
		}

		return biggest.URL, nil
	}

	// If the post has no sample, return an error
	if post.Sample.URL == "" {
		return "", errors.New("post is too large and has no samples")
	}

	// If the post is an image, we just return the sample
	length, err := e.GetContentLength(post.Sample.URL)
	if err != nil {
		return "", err
	}

	// If the sample is too large, return an error
	if length > MAX_POST_SIZE {
		return "", errors.New("sample is too large")
	}

	return post.Sample.URL, nil
}

func (e *E621Service) ParsePost(post *E621PostResponse) (*E621Post, error) {
	if post.ID == 0 {
		return nil, errors.New("post was not found")
	}

	// Find the suitable sample
	url, err := e.FindSuitableSample(post)
	if err != nil {
		return nil, err
	}

	// Get the extension
	parts := strings.Split(url, ".")
	ext := parts[len(parts)-1]

	// Return the post
	return &E621Post{
		ID:   post.ID,
		URL:  url,
		Ext:  ext,
		Size: post.File.Size,
	}, nil
}

func (e *E621Service) GetPopularPosts() ([]*E621Post, error) {
	const url = "https://e621.net/popular.json"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", e.userAgent)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var posts struct {
		Posts []*E621PostResponse `json:"posts"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		return nil, err
	}

	var result []*E621Post
	for _, post := range posts.Posts {
		parsed, err := e.ParsePost(post)
		if err != nil {
			continue
		}

		result = append(result, parsed)
	}

	return result, nil
}
