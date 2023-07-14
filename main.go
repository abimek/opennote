package main

import (
	"cloud.google.com/go/firestore"
	"context"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/abimek/opennote/routing"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
	"sync"
)

var firestoreClient *firestore.Client
var firestoreMutex sync.Mutex

var fireauthClient *auth.Client
var authMutex sync.Mutex

func main() {
	authMutex = sync.Mutex{}
	firestoreMutex = sync.Mutex{}
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
	routing.Route(r, "POST", "/query", openAIQueryEndpointHandler)
	routing.Route(r, "POST", "/api/createEmptyUser", initEmptyUserEndpoint)
	routing.Route(r, "POST", "/api/getUser", getUserEndpoint)
	routing.Route(r, "POST", "/api/updateUser", updateUserEndpoint)
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
