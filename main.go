package main

import (
	"cloud.google.com/go/firestore"
	"context"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/abimek/opennote/routing"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/option"
	"os"
)

var firestoreClient *firestore.Client
var fireauthClient *auth.Client

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	// intialize firebase
	firebaseSetup()
	sessions = map[string]*session{}

	r := gin.Default()
	// cors is not necessary on production, I'll be attempting to run this on a docker container soon
	r.Use(routing.GENERAL)
	// serving static file to OpenAI
	log.Debug().Msg("Initilizing Requests")
	routing.Route(r, "POST", "/api/createEmptyUser", initEmptyUserEndpoint)
	routing.Route(r, "POST", "/api/getUser", getUserEndpoint)
	routing.Route(r, "POST", "/api/updateUser", updateUserEndpoint)
	routing.Route(r, "POST", "/api/validateCredentials", validateCredentials)
	routing.Route(r, "POST", "/messager", queryMessageEndpoint2)
	//steams
	//r.StaticFile("/download/PinePassInstaller", "./resources/content/PinePassInstaller.exe")
	go sessionTimer()
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	r.Run(":" + port)
}

// firebaseSetup inits firebaseAuth and firestore
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
