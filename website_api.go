package main

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"net/http"
)

// RequestErrorResult is the error result when something does go the right way.
type RequestErrorResult struct {
	errorCode WebsiteRequestError `json:"error_code"`
	content   string              `json:"content"`
}

func validateUID(uid string, c *gin.Context) bool {
	valid := validateUIDBool(uid)
	if !valid {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: NonExistentUser,
			content:   "This user does not exist, invalid UID",
		})
	}
	return valid
}

func validateUIDBool(uid string) bool {
	_, err := fireauthClient.GetUser(context.Background(), uid)
	if err != nil {
		return false
	}
	return true
}

// QueryMessageRequest is the request sent to /message when sending a users message to the endpoint.
type QueryMessageRequest struct {
	Uid string `json:"uid"`
	// Chat is the message the user gave the AI
	Chat string `json:"chat"`
}

// queryMessageEndpoint is the endpoint at /message and is the chatMessaging api. If the user exists it starts a chat
// sessions with the openAI bot and enables it to query the users notes.
func queryMessageEndpoint(c *gin.Context) {
	var request QueryMessageRequest
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: InvalidRequestContent,
			content:   "Content doesn't match expected structure",
		})
		return
	}

	sess := GetSessionIfExists(request.Uid)
	if sess == nil {
		if !validateUID(request.Uid, c) {
			return
		}

		docs, err := firestoreClient.Collection("users").Where("Uid", "==", request.Uid).Limit(1).Documents(context.Background()).GetAll()
		if err != nil || len(docs) == 0 {
			c.JSON(http.StatusBadRequest, RequestErrorResult{
				errorCode: FirestoreError,
				content:   "Unable to find user in firestore",
			})
			return
		}

		jsonData, _ := json.Marshal(docs[0].Data())
		var user User
		if err = json.Unmarshal(jsonData, &user); err != nil {
			c.JSON(http.StatusBadRequest, RequestErrorResult{
				errorCode: FirestoreError,
				content:   "Data Format Error",
			})
			return
		}

		sess, err = GetSession(user)
		if err != nil {
			c.JSON(http.StatusExpectationFailed, RequestErrorResult{
				errorCode: InvalidCredsError,
				content:   "Expected valid credentials for user",
			})
			return
		}
	}

	content, err := sess.Message(request.Chat)

	if err != nil {
		sess.userMu.RLock()
		sess.userMu.RUnlock()
		c.JSON(http.StatusExpectationFailed, RequestErrorResult{
			errorCode: InvalidCredsError,
			content:   "Expected valid credentials for user",
		})
		return
	}
	c.String(http.StatusOK, content)
}

// WebsiteCreateUserRequest is the request sent to the /api/createEmptyUser endpoint.
type WebsiteCreateUserRequest struct {
	Uid string `json:"uid"`
}

// initEmptyUserEndpoint is the endpoint at /api/createEmptyUser and it creates a new user in the db if a user doesn't
// exist
func initEmptyUserEndpoint(c *gin.Context) {
	var request WebsiteCreateUserRequest
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: InvalidRequestContent,
			content:   "Content doesn't match expected structure",
		})
		return
	}
	if request.Uid == "" {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: InvalidRequestContent,
			content:   "Empty UID",
		})
		return
	}

	//validate UID as an account
	if !validateUIDBool(request.Uid) {
		return
	}

	user := User{}
	user.Uid = request.Uid

	// upload to firestore (user object) only if it doesn't exist,
	_, _, err := firestoreClient.Collection("users").Add(context.Background(), user)
	if err != nil {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: FirestoreError,
			content:   "Unable to upload document to firestore",
		})
		return
	}
	c.Status(http.StatusCreated)
}

// GetUserRequest is the request sent when trying to get a user at /api/getUser from the frontend.
type GetUserRequest struct {
	Uid string `json:"uid"`
}

// getUserEndpoint is the endpoint at /api/getUser and allows the frontend to get the information about a specific user
// to enable them to populate the screen content about the users information.
func getUserEndpoint(c *gin.Context) {
	var request GetUserRequest
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: InvalidRequestContent,
			content:   "Content doesn't match expected structure",
		})
		return
	}

	if request.Uid == "" {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: InvalidRequestContent,
			content:   "Empty UID",
		})
		return
	}

	if !validateUID(request.Uid, c) {
		return
	}

	docs, err := firestoreClient.Collection("users").Where("Uid", "==", request.Uid).Limit(1).Documents(context.Background()).GetAll()
	if err != nil || len(docs) == 0 {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: FirestoreError,
			content:   "Unable to find user in firestore",
		})
		return
	}

	doc := docs[0]
	c.JSON(http.StatusOK, doc.Data())
	return
}

// updateUserEndpoint is the endpoint at /api/updateUser and allows the frontend to update what a specific user lookslike
func updateUserEndpoint(c *gin.Context) {
	var request User
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: InvalidRequestContent,
			content:   "Content doesn't match expected structure",
		})
		return
	}

	// Update Document
	docs, err := firestoreClient.Collection("users").Where("Uid", "==", request.Uid).Limit(1).Documents(context.Background()).GetAll()
	if err != nil || len(docs) == 0 {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: FirestoreError,
			content:   "Unable to find user in firestore",
		})
		return
	}

	// upload the info for the current session
	if ses := GetSessionIfExists(request.Uid); ses != nil {
		ses.userMu.Lock()
		ses.user = request
		ses.userMu.Unlock()
	}

	doc := docs[0]
	_, err = doc.Ref.Set(context.Background(), request)
	if err != nil {
		log.Error().
			Err(err).
			Str("User", request.Uid).
			Msg("Unable to update firestore")
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: FirestoreError,
			content:   "Unable to Update File in Firestore",
		})
		return
	}
	c.Status(http.StatusOK)
}
