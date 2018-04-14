package main

import (
	"fmt"
	"github.com/zmb3/spotify"
	"net/http"
	"log"
	"os"
)

func redirect(w http.ResponseWriter, r *http.Request) {
	redirectURL := "https://open.spotify.com/user/rooshypooshy/playlist/4jj4dm7CryepjBlKwT4dKe?si=tiUyT3x-QWSJEGBvUEQ7xw"
	clientID    := os.Getenv("CLIENT_ID")
	secretID    := os.Getenv("CLIENT_SECRET")
	fmt.Printf("( 2 ) Client ID: %v\n", clientID)
	auth := spotify.NewAuthenticator(redirectURL, spotify.ScopeUserLibraryRead, spotify.ScopeUserFollowRead)
	auth.SetAuthInfo(clientID, secretID)

	http.Redirect(w, r, auth.AuthURL("state-string"), http.StatusFound)
}

func main() {
	fmt.Println("Starting Botify!")

	testVar := os.Getenv("CLIENT_ID")
	fmt.Printf("( 1 ) Client ID: %v\n", testVar)

	http.HandleFunc("/", redirect )
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Using port: " + port)
	err := http.ListenAndServe(":" + port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}