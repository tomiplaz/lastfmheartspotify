package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

const (
	authorizeURL           = "https://accounts.spotify.com/authorize"
	scopeUserLibraryModify = "user-library-modify"
)

func main() {
	http.HandleFunc("/login", login)

	fmt.Println("Serving on :8080")
	http.ListenAndServe(":8080", nil)
}

func login(w http.ResponseWriter, req *http.Request) {
	params, err := getAuthParams()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	url := fmt.Sprintf("%s?%s", authorizeURL, params)
	http.Redirect(w, req, url, http.StatusSeeOther)
}

func getAuthParams() (string, error) {
	bytes := make([]byte, 0, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}

	redirectURI := "https://localhost:8080/success"
	state := base64.StdEncoding.EncodeToString(bytes)
	scope := strings.Join([]string{scopeUserLibraryModify}, " ")

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
