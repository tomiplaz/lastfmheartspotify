package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"

	"github.com/joho/godotenv"
)

const (
	authorizeURL      = "https://accounts.spotify.com/authorize"
	apiTokenURL       = "https://accounts.spotify.com/api/token"
	redirectURI       = "https://127.0.0.1:8443/callback"
	userLibraryRead   = "user-library-read"
	userLibraryModify = "user-library-modify"
)

// TODO: Instantiate a request to track instead of it being global
var state string

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalln("error loading .env file")
	}

	http.HandleFunc("/", index)
	http.HandleFunc("/login", login)
	http.HandleFunc("/callback", callback)

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

	fmt.Println("access token received:", resParsed.AccessToken)
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
