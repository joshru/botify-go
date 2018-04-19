package main

import (
	"fmt"
	"github.com/zmb3/spotify"
	"net/http"
	"log"
	"os"
	"github.com/sha1sum/golang_groupme_bot/bot"
	"context"
)
//Random track ID: 1IruBrVHO0XS9SfXGoYBXn
//playlist ID: 4jj4dm7CryepjBlKwT4dKe

//const redirectURL = "https://open.spotify.com/user/rooshypooshy/playlist/4jj4dm7CryepjBlKwT4dKe?si=tiUyT3x-QWSJEGBvUEQ7xw"
//const redirectURL = "http://localhost:8080"
const redirectURL = "https://groupme-botify.herokuapp.com"

var (
	clientID    = os.Getenv("CLIENT_ID")
	secretID    = os.Getenv("CLIENT_SECRET")
	stateString = "groupme_bot_state"
	ch          = make(chan *spotify.Client)
	auth = spotify.NewAuthenticator(redirectURL, spotify.ScopeUserReadPrivate, spotify.ScopeUserLibraryRead, spotify.ScopePlaylistModifyPublic)
)

type Handler struct{}

func (handler Handler) Handle(term string, c chan []*bot.OutgoingMessage, message bot.IncomingMessage) {
	fmt.Println("Handler called!")
	// exit early if the received message was posted by a bot
	if message.SenderType == "bot" {
		return
	}
	fmt.Println("Found message:", message.Text)
}

// Begin Spotify authorization flow, after user logs in they will be redirected to a success page
// https://godoc.org/github.com/zmb3/spotify#Authenticator
func completeAuth(w http.ResponseWriter, r *http.Request) {
	token, err := auth.Token(stateString, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
	}
	if st := r.FormValue("state"); st != stateString {
		http.NotFound(w, r)
		log.Fatalf("State mismatch %s != %s\n", st, stateString)
	}
	client := auth.NewClient(token)

	//fmt.Fprintf(w, "Login completed!")
	fmt.Println("Login completed!")
	ch <- &client
}
//https://open.spotify.com/track/6dHatCnuOb1TdBIeJTK3Y0?si=V_PGrzUEQy2BXNZGY33YnA
func main() {
	fmt.Println("Starting Botify!")

	// fetch port from Heroku, or use 8080 if no port environment variable is set
	//port := os.Getenv("PORT")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	auth.SetAuthInfo(clientID, secretID)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		completeAuth(w, r)
		http.ServeFile(w, r, "./index.html")
	} )
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./favicon.ico")
	} )
	//go http.ListenAndServe(":" + port, nil)
	srv := &http.Server{Addr: ":" + port}
	go srv.ListenAndServe()

	url := auth.AuthURL(stateString)
	fmt.Println("Please log in to Spotify via:", url)

	// wait for auth to complete
	client := <- ch
	user, err := client.CurrentUser()
	if err != nil {
		log.Fatal(err)
	}
	client.AddTracksToPlaylist("rooshypooshy", "4jj4dm7CryepjBlKwT4dKe", "1IruBrVHO0XS9SfXGoYBXn")
	fmt.Println("You are logged in as:", user.ID)

	srv.Shutdown(context.Background())

	fmt.Println("Creating groupme bot")
	commands := make([]bot.Command, 0)
	songs := bot.Command{
		Triggers: []string {
			"https://open.spotify",
			"testing",
			"",
		},
		Handler: new(Handler),
		BotID: "d01b6e91b7c35b66405ba58dbf",
	}
	commands = append(commands, songs)
	go bot.Listen(commands)

	// block forever
	select {}
}