package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

const (
	authorizeURL = "https://accounts.spotify.com/authorize"
	apiTokenURL  = "https://accounts.spotify.com/api/token"

	redirectURI = "https://127.0.0.1:8443/callback"

	userLibraryRead   = "user-library-read"
	userLibraryModify = "user-library-modify"

	searchURL = "https://api.spotify.com/v1/search"
)

// TODO: Instantiate singletons instead of having globals
var (
	state       string
	accessToken string
)

type SearchResponse struct {
	Tracks struct {
		Items []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Artists []struct {
				Name string `json:"name"`
			} `json:"artists"`
		} `json:"items"`
	} `json:"tracks"`
}

type SpotifyTrack struct {
	ID     string
	Name   string
	Artist string
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalln("error loading .env file")
	}

	http.HandleFunc("/", index)
	http.HandleFunc("/login", login)
	http.HandleFunc("/callback", callback)
	http.HandleFunc("/foo", foo)

	fmt.Println("starting server on https://127.0.0.1:8443")

	err := http.ListenAndServeTLS("127.0.0.1:8443", "certs/server.crt", "certs/server.key", nil)
	if err != nil {
		fmt.Println("error starting server:", err)
		return
	}
}

func index(w http.ResponseWriter, req *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func login(w http.ResponseWriter, req *http.Request) {
	params, err := getAuthParams()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	url := fmt.Sprintf("%s?%s", authorizeURL, params)
	http.Redirect(w, req, url, http.StatusSeeOther)
}

func callback(w http.ResponseWriter, req *http.Request) {
	params := req.URL.Query()

	if params.Get("error") != "" {
		fmt.Println("callback error:", params.Get("error"))
		http.Error(w, fmt.Sprintf("callback error: %s", params.Get("error")), http.StatusBadRequest)
		return
	}

	if params.Get("state") != state {
		fmt.Println("invalid state value:", params.Get("state"))
		http.Error(w, "invalid state value", http.StatusBadRequest)
		return
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", params.Get("code"))
	data.Set("redirect_uri", redirectURI)

	reqBody := strings.NewReader(data.Encode())
	req, err := http.NewRequestWithContext(req.Context(), "POST", apiTokenURL, reqBody)
	if err != nil {
		fmt.Println("error creating token request:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	authStr := fmt.Sprintf("%s:%s", clientID, clientSecret)
	authStrEncoded := base64.StdEncoding.EncodeToString([]byte(authStr))

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", authStrEncoded))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("error sending token request:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resBytes, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("error reading token response body:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if res.StatusCode != 200 {
		fmt.Println("token request error:", string(resBytes))
		http.Error(w, fmt.Sprintf("token request err: %s", string(resBytes)), http.StatusInternalServerError)
		return
	}

	var resParsed struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(resBytes, &resParsed); err != nil {
		http.Error(w, fmt.Sprintf("%s: %s", err.Error(), string(resBytes)), http.StatusInternalServerError)
		return
	}

	accessToken = resParsed.AccessToken

	http.Redirect(w, req, "/foo", http.StatusSeeOther)
}

func foo(w http.ResponseWriter, req *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/foo.html"))

	lastFmApiKey := os.Getenv("LAST_FM_API_KEY")
	if lastFmApiKey == "" {
		log.Fatalln("LAST_FM_API_KEY not set")
	}

	lastFmUser := os.Getenv("LAST_FM_USER")
	if lastFmApiKey == "" {
		log.Fatalln("LAST_FM_USER not set")
	}

	lastFm, err := InitLastFm(lastFmUser, lastFmApiKey)
	if err != nil {
		log.Fatalln("error initializing LastFm struct: " + err.Error())
	}

	lovedTracks, err := lastFm.GetLovedTracks(true)
	if err != nil {
		log.Fatalln("error getting loved tracks: " + err.Error())
	}

	count := 3
	randIndeces := make([]int, 0, count)
	for range count {
		randIndeces = append(randIndeces, rand.Intn(len(lovedTracks)))
	}

	lastFmTracks := make([]LovedTrack, 0, len(randIndeces))
	spotifyTracks := make([]SpotifyTrack, 0, len(randIndeces))
	for _, i := range randIndeces {
		t := lovedTracks[i]
		lastFmTracks = append(lastFmTracks, t)
		spotifyTrack, err := doSearchRequest(req.Context(), t.Artist.Name, t.Name)
		if err != nil {
			fmt.Println("search spotify track:", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		spotifyTracks = append(spotifyTracks, spotifyTrack)
	}

	data := struct {
		LastFmTracks  []LovedTrack
		SpotifyTracks []SpotifyTrack
	}{
		LastFmTracks:  lastFmTracks,
		SpotifyTracks: spotifyTracks,
	}

	if err := tmpl.ExecuteTemplate(w, "foo.html", data); err != nil {
		fmt.Println("foo template error:", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getAuthParams() (string, error) {
	bytes := make([]byte, 0, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}

	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	state = base64.StdEncoding.EncodeToString(bytes)
	scope := strings.Join([]string{userLibraryRead, userLibraryModify}, " ")

	params := map[string]string{
		"client_id":     clientID,
		"response_type": "code",
		"redirect_uri":  redirectURI,
		"state":         state,
		"scope":         scope,
	}

	slice := make([]string, 0, len(params))
	for k, v := range params {
		slice = append(slice, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(slice, "&"), nil
}

func doSearchRequest(ctx context.Context, artist string, track string) (SpotifyTrack, error) {
	endpoint, err := url.Parse(searchURL)
	if err != nil {
		return SpotifyTrack{}, fmt.Errorf("parse search url: %w", err)
	}

	params := url.Values{}
	params.Set("q", fmt.Sprintf("artist:%s track:%s", artist, track))
	params.Set("type", "track")
	endpoint.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint.String(), nil)
	if err != nil {
		return SpotifyTrack{}, fmt.Errorf("create search request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return SpotifyTrack{}, fmt.Errorf("error sending search request: %w", err)
	}

	resBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return SpotifyTrack{}, fmt.Errorf("error reading search response body: %w", err)
	}

	if res.StatusCode != 200 {
		return SpotifyTrack{}, fmt.Errorf("search request error: %s", string(resBytes))
	}

	var resParsed SearchResponse
	if err := json.Unmarshal(resBytes, &resParsed); err != nil {
		return SpotifyTrack{}, fmt.Errorf("unmarshal search response: %w", err)
	}

	if len(resParsed.Tracks.Items) == 0 {
		fmt.Printf("no track found for %s - %s\n", artist, track)
		return SpotifyTrack{}, nil
	}

	t := resParsed.Tracks.Items[0]
	return SpotifyTrack{
		ID:     t.ID,
		Name:   t.Name,
		Artist: t.Artists[0].Name,
	}, nil
}
