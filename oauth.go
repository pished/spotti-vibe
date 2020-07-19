package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var BearToken, RefreshToken string
var server *http.Server
var spotID = os.Getenv("SPOTIFY_CLIENT_ID")

type Tokens struct {
	Access_token  string
	Token_type    string
	Expires_in    int
	Refresh_token string
	Scope         string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}

}

func basicAuth64(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}

func redirectPolicyFunc(req *http.Request, via []*http.Request) error {
	req.Header.Add("Authorization", basicAuth64(os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET")))
	return nil
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	log.Print("Pinged webpage.")
	if k, ok := r.URL.Query()["code"]; ok {
		client := &http.Client{
			CheckRedirect: redirectPolicyFunc,
		}
		payload := url.Values{}
		payload.Set("grant_type", "authorization_code")
		payload.Set("code", strings.Join(k, ""))
		payload.Set("redirect_uri", "http://localhost:8080")
		req, err := http.NewRequest(
			"POST",
			"https://accounts.spotify.com/api/token",
			strings.NewReader(payload.Encode()),
		)
		check(err)
		req.Header.Set("Content-type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", basicAuth64(os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET")))
		fmt.Println(req.Header.Get("Authorization"))
		reqDump, _ := httputil.DumpRequest(req, true)
		log.Println(string(reqDump))
		resp, err := client.Do(req)
		check(err)
		defer resp.Body.Close()
		var tokens Tokens
		err = json.NewDecoder(resp.Body).Decode(&tokens)
		check(err)
		log.Printf("%+v\n", tokens)
	} else {
		log.Printf("User denied access to Spotify!")
	}
	server.Shutdown(context.Background())
}

func AuthenticateSpotify() {
	f, err := os.Create("log.txt")
	check(err)
	defer f.Close()

	log.SetOutput(io.MultiWriter(f, os.Stdout))

	redirect := url.QueryEscape("http://localhost:8080")
	scopes := url.QueryEscape("playlist-read-collaborative playlist-read-private user-read-currently-playing")
	initURL := fmt.Sprintf(
		"https://accounts.spotify.com/authorize?client_id=%v&response_type=code&redirect_uri=%v&scope=%v&state=testing",
		spotID,
		redirect,
		scopes,
	)

	openbrowser(initURL)

	server = &http.Server{Addr: ":8080"}
	http.HandleFunc("/", handleRoot)
	server.ListenAndServe()
	// Setup signal Capture

	log.Printf("After server")
}
