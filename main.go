package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "regexp"
    "strings"
    "github.com/bwmarrin/discordgo"
    "github.com/zmb3/spotify"
)

const redirectURL = "http://botify.sudont.org:8080"

type song struct {
    trackURL   string
    playlistID spotify.ID
}

var (
    clientID    = os.Getenv("CLIENT_ID")
    secretID    = os.Getenv("CLIENT_SECRET")
    stateString = "groupme_bot_state"
    userID      = "rooshypooshy"
    ch          = make(chan *spotify.Client)
    gmChan      = make(chan *song)
    auth        = spotify.NewAuthenticator(redirectURL, spotify.ScopeUserReadPrivate, spotify.ScopeUserLibraryRead, spotify.ScopePlaylistModifyPublic)
)

func getPlaylistID(guildID string) spotify.ID {
    // main chat
    if guildID == "322958610068144132" {
        return spotify.ID("2KnpXXFuYrf9zEItCMaQAd")

        // underwater rocket squad
    } else if guildID == "654773487155544068" {
        return spotify.ID("6K1wQP7FDAwJKN6aM4TDL1")

    } else {
        fmt.Println("Failed to find playlist for Guild ID: {}", guildID)
        return spotify.ID("")
    }
}

func handle(s *discordgo.Session, m *discordgo.MessageCreate) {

    // Ignore any messages that come from the bot itself
    if m.Author.ID == s.State.User.ID {
        return
    }

    if m.Content == "!old" && m.GuildID == "322958610068144132" {
        s.ChannelMessageSend(m.ChannelID, "https://open.spotify.com/user/rooshypooshy/playlist/4jj4dm7CryepjBlKwT4dKe")

    } else if m.Content == "!playlist" {
        if m.GuildID == "322958610068144132" {
            s.ChannelMessageSend(m.ChannelID, "https://open.spotify.com/playlist/2KnpXXFuYrf9zEItCMaQAd?si=CRyzpVbDTGiH9iCP57g8uw")
        } else if m.GuildID == "654773487155544068" {
            s.ChannelMessageSend(m.ChannelID, "https://open.spotify.com/playlist/6K1wQP7FDAwJKN6aM4TDL1?si=z2FGEYOFSxGQz5o9F2fEaA")
        }

    }  else if (m.Author.ID == "111360053654638592") {
        s.MessageReactionAdd(m.ChannelID, m.ID, "👎")
    } else if strings.Contains(m.Content, "open.spotify.com/track") {
        fmt.Println("Found spotify link, handling...")
        postedSong := new(song)
        postedSong.trackURL = m.Content
        postedSong.playlistID = getPlaylistID(m.GuildID)
        gmChan <- postedSong
        s.MessageReactionAdd(m.ChannelID, m.ID, "👍")
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
    endToken := "?si="
    // track id regex: track/(.*?)?si=
    matcher := regexp.MustCompile("track/(.*?)?si=")
    matchedStr := matcher.FindString(url)
    trimmed := matchedStr[len(startToken) : len(matchedStr)-len(endToken)]
    return trimmed
}

// checks for duplicates in the playlist. True if duplicate, false otherwise
func checkForDuplicates(track *spotify.FullTrack, playlist []spotify.PlaylistTrack) bool {
    for _, element := range playlist {
        if element.Track.Name == track.Name {
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
        postedSong := <-gmChan

        foundTrack := spotify.ID(trackTrimmer(postedSong.trackURL))
        trackID := trackTrimmer(postedSong.trackURL)
        trackObj, err := client.GetTrack(spotify.ID(trackID))
        if err != nil {
            fmt.Println("Unable to locate track: {}, {}", trackID, err)
        }

        fmt.Println("Found track:", trackObj.SimpleTrack.Name)

        playlistTracks, _ := client.GetPlaylistTracks(postedSong.playlistID)
        isRepost := checkForDuplicates(trackObj, playlistTracks.Tracks)

        if !isRepost {
            client.AddTracksToPlaylist(postedSong.playlistID, foundTrack)
        }
    }
}

//https://open.spotify.com/track/6dHatCnuOb1TdBIeJTK3Y0?si=V_PGrzUEQy2BXNZGY33YnA
func main() {
    fmt.Println("Starting Botify!")

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    auth.SetAuthInfo(clientID, secretID)

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        completeAuth(w, r)
        http.ServeFile(w, r, "./index.html")
    })
    http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "./favicon.ico")
    })

    // Start http server for authentication redirect
    srv := &http.Server{Addr: ":" + port}
    go srv.ListenAndServe()

    url := auth.AuthURL(stateString)
    fmt.Println("Please log in to Spotify via:", url)

    // channel will wait for auth to complete
    go addTrackToPlaylist(<-ch)

    srv.Shutdown(context.Background())

    botID := os.Getenv("BOT_ID")
    if botID == "" {
        fmt.Println("Unable to get Bot_ID!! Exiting...")
        return
    }

    fmt.Println("Creating discord bot")
    bot, err := discordgo.New("Bot " + botID)
    if err != nil {
        fmt.Println("Error creating Discord session, ", err)
        return
    }

    // add message handler callback
    bot.AddHandler(handle)
    err = bot.Open()
    if err != nil {
        fmt.Println("Error opening connection, ", err)
        return
    }
    fmt.Println("Bot is now running...")

    // block forever (not sure if this is necessary)
    select {}
}
