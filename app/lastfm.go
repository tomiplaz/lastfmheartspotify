package main

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type LastFm struct {
	user   string
	apiKey string
}

type LovedTrack struct {
	Artist struct {
		Name string `json:"name"`
	} `json:"artist"`
	Name string `json:"name"`
}

type LovedTracksResponse struct {
	LovedTracks struct {
		Track []LovedTrack `json:"track"`
		Attr  struct {
			TotalPages string `json:"totalPages"`
		} `json:"@attr"`
	} `json:"lovedtracks"`
}

func InitLastFm(user string, apiKey string) (LastFm, error) {
	return LastFm{
		user:   user,
		apiKey: apiKey,
	}, nil
}

// fetchPage retrieves a single page of loved tracks from Last.fm API
// It returns the tracks for that page, the total number of pages, and any error encountered
func (lastFm LastFm) fetchPage(page int, params map[string]string, progressCh chan<- int) ([]LovedTrack, int, error) {
	// Create a copy of params to avoid modifying the original
	pageParams := make(map[string]string)
	maps.Copy(pageParams, params)

	pageParams["page"] = fmt.Sprintf("%d", page)

	url := fmt.Sprintf("http://ws.audioscrobbler.com/2.0?%s", getQueryStr(pageParams))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Add("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("do request: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, 0, fmt.Errorf("read response body: %w", err)
	}

	var parsed LovedTracksResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, 0, fmt.Errorf("parse json: %w", err)
	}

	totalPages, err := strconv.Atoi(parsed.LovedTracks.Attr.TotalPages)
	if err != nil {
		return nil, 0, fmt.Errorf("parse total pages: %w", err)
	}

	if progressCh != nil {
		progressCh <- page
	}

	return parsed.LovedTracks.Track, totalPages, nil
}

func (lastFm LastFm) GetLovedTracks(firstOnly bool) ([]LovedTrack, error) {
	maxCon := 10
	params := map[string]string{
		"api_key": lastFm.apiKey,
		"user":    lastFm.user,
		"method":  "user.getlovedtracks",
		"format":  "json",
	}

	// Create a progress channel
	progressCh := make(chan int, maxCon)

	// Start a goroutine to handle progress reporting
	var totalPages int
	go func() {
		completed := []int{1}
		percentage := 0
		for page := range progressCh {
			completed = append(completed, page)
			// Only print if we know the total
			if totalPages > 0 {
				percentage = int(float64(len(completed)) / float64(totalPages) * 100)
				if percentage <= 100 {
					fmt.Printf("\rProgress: %d%s", percentage, "%")
				}
				if percentage == 100 {
					fmt.Println()
				}
			}
		}
	}()

	// Fetch page 1 to determine the total number of pages
	firstPageTracks, pages, err := lastFm.fetchPage(1, params, progressCh)
	if err != nil {
		close(progressCh)
		return nil, err
	}
	totalPages = pages

	// Initialize result slice with first page results
	allTracks := make([]LovedTrack, 0, totalPages*50)
	allTracks = append(allTracks, firstPageTracks...)

	// If there's only one page, we're done
	if totalPages <= 1 || firstOnly {
		close(progressCh)
		fmt.Printf("Total tracks: %d\n", len(allTracks))
		return allTracks, nil
	}

	var pagesWg sync.WaitGroup
	var fetchErr error
	var tracksMu, fetchErrMu sync.Mutex

	// Semaphore to allow max page fetching goroutines
	pageSem := make(chan struct{}, maxCon)

	// Start goroutines for the remaining pages
	for page := 2; page <= totalPages; page++ {
		// Reassign to prevent race condition
		p := page

		// Start goroutine to fetch page
		pagesWg.Go(func() {
			// Acquire semaphore (blocks if max)
			pageSem <- struct{}{}
			// Release semaphore when done
			defer func() { <-pageSem }()

			// Fetch page
			tracks, _, err := lastFm.fetchPage(p, params, progressCh)
			if err != nil {
				// Save the first error we encounter
				fetchErrMu.Lock()
				if fetchErr == nil {
					fetchErr = fmt.Errorf("error fetching page %d: %w", p, err)
				}
				fetchErrMu.Unlock()
				return
			}

			// Safely append tracks
			tracksMu.Lock()
			allTracks = append(allTracks, tracks...)
			tracksMu.Unlock()
		})
	}

	pagesWg.Wait()

	close(progressCh)

	if fetchErr != nil {
		return nil, fetchErr
	}

	fmt.Printf("Total tracks: %d\n", len(allTracks))

	return allTracks, nil
}

func getQueryStr(m map[string]string) string {
	pairs := make([]string, 0, len(m))
	for k, v := range m {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(pairs, "&")
}
