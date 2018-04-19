// Package bot is used to listen for incoming GroupMe bot callbacks and parse the text for any matching commands
// that may be present and then handle the commands.
package bot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"

	"time"
)

// Commands houses a list of possible bot commands to use when parsing incoming bot messages
var commands []Command

// Command is a way of indicating a trigger and keyword to determine whether incoming text should be handled by a Handler
type Command struct {
	// Triggers is a list of terms used to determine whether a message should be treated as a bot command or not. The
	// triggers will be lowercased when matching so are case insensitive.
	Triggers []string
	// Handler is the bot handler to use when either of the Triggers are present
	Handler Handler
	// BotID is the ID of the GroupMe bot to use for posting replies to commands
	BotID string
}

// handler will take an incoming HTTP request and treat it as a POST request from a GroupMe bot and then fire off the
// handle function as a goroutine.
func handler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		fmt.Println("Handling request...")
		decoder := json.NewDecoder(request.Body)
		var post IncomingMessage
		err := decoder.Decode(&post)
		if err != nil {
			fmt.Println(err)
		}
		for _, c := range commands {
			for _, t := range c.Triggers {
				var term string

				if strings.Contains(strings.ToLower(post.Text), strings.ToLower(t)) {
					term = strings.Replace(strings.ToLower(post.Text), " "+t+" ", "", -1)
					term = strings.Replace(term, t+" ", "", -1)
					term = strings.Replace(term, " "+t, "", -1)
					go handle(strings.Trim(term, " "), c, post)
					h := &c
					v := reflect.ValueOf(h.Handler).Elem()
					v.Set(reflect.Zero(v.Type()))
				}
			}
		}
	})
}

// search takes a given search term and queries uses the searcher to find the term, and then
// posts the message returned from the searcher using PostMessage.
func handle(term string, command Command, message IncomingMessage) {
	fmt.Println("Handling term \"" + term + "\".")

	c := make(chan []*OutgoingMessage, 1)
	go command.Handler.Handle(term, c, message)
	m := <-c
	for _, v := range m {
		if v.Err != nil {
			_, err := PostMessage(&OutgoingMessage{Text: fmt.Sprint(v.Err)}, command.BotID)
			if err != nil {
				fmt.Println(err)
			}
			return
		}
		_, err := PostMessage(v, command.BotID)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Outgoing message: %+v", v)
		time.Sleep(time.Second)
	}
}

// port determines the port to listen on as declared by the "PORT" environment variable, or uses 80 if the environment
// variable is not defined.
func port() string {
	var port = os.Getenv("PORT")
	if port == "" {
		port = "80"
	}
	fmt.Println("Using port", port)
	return ":" + port
}

// Listen will start an HTTP server and begin listening for bot commands
func Listen(c []Command) {
	commands = c
	mux := http.NewServeMux()
	mux.Handle("/", handler())
	fmt.Println("HTTP handler set. Listening.")
	err := http.ListenAndServe(port(), mux)
	if err != nil {
		fmt.Println(err)
	}
}
