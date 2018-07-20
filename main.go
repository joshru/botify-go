package main

import (
	"fmt"
	"github.com/zmb3/spotify"
	"net/http"
	"log"
	"os"
	"github.com/sha1sum/golang_groupme_bot/bot"
	"context"
	"regexp"
)

const redirectURL = "http://botify.sudont.org:8080"

var (
	botID		= "d1751969b118bd1dac42bdd5c5"
	clientID    = os.Getenv("CLIENT_ID")
	secretID    = os.Getenv("CLIENT_SECRET")
	stateString = "groupme_bot_state"
	userID		= "rooshypooshy"
	playlistID  = spotify.ID("4jj4dm7CryepjBlKwT4dKe")
	ch          = make(chan *spotify.Client)
	gmChan		= make(chan string)
	auth = spotify.NewAuthenticator(redirectURL, spotify.ScopeUserReadPrivate, spotify.ScopeUserLibraryRead, spotify.ScopePlaylistModifyPublic)
)

// default Handler instance is sufficient for handling posts to groupme
type Handler struct{}

// implement Handle function for Handler interface
func (handler Handler) Handle(term string, c chan []*bot.OutgoingMessage, message bot.IncomingMessage) {
	// exit early if the received message was posted by a bot
	if message.SenderType == "bot" {
		return
	}
	fmt.Println("Handler found message:", message.Text)

	if message.Text == "!playlist" {
		// post the playlist to the group
		postPlaylist()
		fmt.Println("Posting playlist to group.")
	} else {
		// write message to channel so it can be seen by the track adding function
		gmChan <- message.Text
	}
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

	// log completion and write the client object to global channel
	fmt.Println("Login completed!")
	ch <- &client
}

// trim the handled URL to extract song ID
func trackTrimmer(url string) string {
	startToken := "track/"
	endToken :="?si="
	// track id regex: track/(.*?)?si=
	matcher := regexp.MustCompile("track/(.*?)?si=")
	matchedStr := matcher.FindString(url)
	trimmed := matchedStr[len(startToken):len(matchedStr) - len(endToken)]
	return trimmed
}

// checks for duplicates in the playlist. True if duplicate, false otherwise
func checkForDuplicates(track *spotify.FullTrack, playlist []spotify.PlaylistTrack) bool {
	for _, element := range playlist {
		if element.Track.Name == track.Name {
			postText("Reeeepost")
			fmt.Println(track.Name + " is a repost, ignoring")
			return true
		}
	}
	return false
}

// adds the linked track to the playlist
func addTrackToPlaylist(client *spotify.Client) {
	// infinite loop runs in separate goroutine
	for {
		trackURL := <- gmChan

		foundTrack := spotify.ID(trackTrimmer(trackURL))
		trackID := trackTrimmer(trackURL)
		trackObj, err := client.GetTrack(spotify.ID(trackID))
		fmt.Println("Found track:", trackObj.SimpleTrack.Name)
		if err != nil {
			fmt.Println("Unable to locate track:", trackID)
		}

		playlistTracks, _ := client.GetPlaylistTracks(userID, playlistID)
		isUnique := checkForDuplicates(trackObj, playlistTracks.Tracks)

		if isUnique {
			client.AddTracksToPlaylist(userID, playlistID, foundTrack)
		}

	}
}

// posts a link to the playlist
func postPlaylist() {
	msg := []*bot.OutgoingMessage{{Text: "https://open.spotify.com/user/rooshypooshy/playlist/4jj4dm7CryepjBlKwT4dKe"}}
	bot.PostMessage(msg[0], botID)
}

func postText(m string) {
	msg := []*bot.OutgoingMessage{{Text: m}}
	bot.PostMessage(msg[0], botID)
}

//https://open.spotify.com/track/6dHatCnuOb1TdBIeJTK3Y0?si=V_PGrzUEQy2BXNZGY33YnA
func main() {
	fmt.Println("Starting Botify!")

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
	go addTrackToPlaylist(<- ch)

	srv.Shutdown(context.Background())

	fmt.Println("Creating groupme bot")
	commands := make([]bot.Command, 0)
	songs := bot.Command{
		Triggers: []string {
			"https://open.spotify.com/track",
			"!playlist",
		},
		Handler: new(Handler),
		BotID: botID,
	}
	commands = append(commands, songs)
	bot.Listen(commands)

	// block forever (not sure if this is still necessary)
	select {}
}
