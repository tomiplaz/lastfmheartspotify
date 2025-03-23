package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
		Track []LovedTrack
	} `json:"lovedtracks"`
}

func InitLastFm(user string, apiKey string) (LastFm, error) {
	return LastFm{
		user:   user,
		apiKey: apiKey,
	}, nil
}

func (lastFm LastFm) GetLovedTracks() ([]LovedTrack, error) {
	params := map[string]string{
		"api_key": lastFm.apiKey,
		"user":    lastFm.user,
		"method":  "user.getlovedtracks",
		"format":  "json",
	}
	url := fmt.Sprintf("http://ws.audioscrobbler.com/2.0?%s&page=1", getQueryStr(params))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Add("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var parsed LovedTracksResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}

	return parsed.LovedTracks.Track, nil
}

func getQueryStr(m map[string]string) string {
	pairs := make([]string, 0, len(m))
	for k, v := range m {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(pairs, "&")
}
