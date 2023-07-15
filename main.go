package main

import (
	"cloud.google.com/go/firestore"
	"context"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/abimek/opennote/routing"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/option"
)

// Handle the chance that they need mutexes if that actually happens
var firestoreClient *firestore.Client

var fireauthClient *auth.Client

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	// intialize firebase
	firebaseSetup()

	r := gin.Default()
	// cors is not necessary on production, I'll be attempting to run this on a docker container soon
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3323", "https://chat.openai.com"}
	r.Use(routing.CORS)

	// serving static file to OpenAI
	r.StaticFile("/.well-known/ai-plugin.json", "./resources/ai-plugin.json")
	r.StaticFile("/.well-known/logo.png", "./resources/logo.png")
	r.StaticFile("/.well-known/openapi.yaml", "./resources/openapi.yaml")
	log.Debug().Msg("Initilizing Requests")
	routing.Route(r, "POST", "/api/createEmptyUser", initEmptyUserEndpoint)
	routing.Route(r, "POST", "/api/getUser", getUserEndpoint)
	routing.Route(r, "POST", "/api/updateUser", updateUserEndpoint)
	routing.Route(r, "POST", "/message", queryMessageEndpoint)

	log.Info().Msgf("OpenNote server running on port %s", "3323")
	r.Run(":3323")
}

// firebaseSetup will intilizes the firestore and fireauth clients
func firebaseSetup() {
	// intialize firestore
	config := &firebase.Config{
		StorageBucket: "todo",
	}
	opt := option.WithCredentialsFile("resources/firebase/key.json")
	app, err := firebase.NewApp(context.Background(), config, opt)
	if err != nil {
		panic("Unablet to connect to firebase")
	}

	firestoreClient, err = app.Firestore(context.Background())
	if err != nil {
		panic("Unable to connect to firebase storage")
	}

	// initilize fireauth
	fireauthClient, err = app.Auth(context.Background())
	if err != nil {
		panic("Unable to conenct to fireauth")
	}
}
