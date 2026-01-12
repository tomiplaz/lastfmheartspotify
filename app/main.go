package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	authorizeURL      = "https://accounts.spotify.com/authorize"
	apiTokenURL       = "https://accounts.spotify.com/api/token"
	redirectURI       = "http://localhost:8080/callback"
	userLibraryRead   = "user-library-read"
	userLibraryModify = "user-library-modify"
)

// TODO: Instantiate a request to track instead of it being global
var state string

func main() {
	http.HandleFunc("/login", login)
	http.HandleFunc("/callback", callback)

	fmt.Println("starting server on :443")

	err := http.ListenAndServeTLS(":443", "certs/server.crt", "certs/server.key", nil)
	if err != nil {
		fmt.Println("error starting server:", err)
		return
	}
}

func login(w http.ResponseWriter, req *http.Request) {
	params, err := getAuthParams()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	url := fmt.Sprintf("%s?%s", authorizeURL, params)
	http.Redirect(w, req, url, http.StatusSeeOther)
}

func callback(w http.ResponseWriter, req *http.Request) {
	params := req.URL.Query()

	if params.Get("error") != "" {
		fmt.Println("callback error:", params.Get("error"))
		return
	}

	if params.Get("state") != state {
		fmt.Println("invalid state value:", params.Get("state"))
		return
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", params.Get("code"))
	data.Set("redirect_uri", redirectURI)

	body := strings.NewReader(data.Encode())
	tokenReq, err := http.NewRequestWithContext(req.Context(), "POST", apiTokenURL, body)
	if err != nil {
		fmt.Println("error creating token request:", err)
		return
	}

	clientID := ""
	clientSecret := ""
	authStr := fmt.Sprintf("%s:%s", clientID, clientSecret)
	authStrEncoded := base64.StdEncoding.EncodeToString([]byte(authStr))

	tokenReq.Header.Set("Authorization", fmt.Sprintf("Basic %s", authStrEncoded))
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	tokenRes, err := http.DefaultClient.Do(tokenReq)
	if err != nil {
		fmt.Println("error sending token request:", err)
	}
}

func getAuthParams() (string, error) {
	bytes := make([]byte, 0, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}

	state = base64.StdEncoding.EncodeToString(bytes)
	scope := strings.Join([]string{userLibraryRead, userLibraryModify}, " ")

	params := map[string]string{
		"client_id":     "123",
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
