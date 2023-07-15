package main

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"net/http"
)

type RequestErrorResult struct {
	errorCode WebsiteRequestError `json:"error_code"`
	content   string              `json:"content"`
}

func validateUID(uid string, c *gin.Context) bool {
	_, err := fireauthClient.GetUser(context.Background(), uid)
	if err != nil {
		log.Error().
			Err(err).
			Str("User", uid).
			Msg("Invalid User")
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: NonExistentUser,
			content:   "This user does not exist, invalid UID",
		})
		return false
	}
	return true
}

func validateUIDBool(uid string) bool {
	_, err := fireauthClient.GetUser(context.Background(), uid)
	if err != nil {
		log.Error().
			Err(err).
			Str("User", uid).
			Msg("Failed to validate UID, database error")
		return false
	}
	return true
}

type QueryMessageRequest struct {
	Uid  string `json:"uid"`
	Chat string `json:"chat"`
}

func queryMessageEndpoint(c *gin.Context) {
	var request QueryMessageRequest
	if err := c.BindJSON(&request); err != nil {
		log.Error().
			Err(err).
			Msg("Invalid json data on message endpoint")
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

	sess := GetSessionIfExists(request.Uid)
	if sess == nil {

		if !validateUID(request.Uid, c) {
			return
		}

		docs, err := firestoreClient.Collection("users").Where("Uid", "==", request.Uid).Limit(1).Documents(context.Background()).GetAll()
		if err != nil || len(docs) == 0 {
			log.Debug().Str("User", request.Uid).Msg("Request trying to find invalid user")
			c.JSON(http.StatusBadRequest, RequestErrorResult{
				errorCode: FirestoreError,
				content:   "Unable to find user in firestore",
			})
			return
		}
		jsonData, _ := json.Marshal(docs[0].Data())
		var user User
		if err = json.Unmarshal(jsonData, &user); err != nil {
			log.Error().
				Err(err).
				Str("User", user.Uid).
				Str("Content", string(jsonData)).
				Msg("Invalid json data on /message endpoint")
			c.JSON(http.StatusBadRequest, RequestErrorResult{
				errorCode: FirestoreError,
				content:   "Data Format Error",
			})
			return
		}
		sess, err = GetSession(user)
		if err != nil {
			log.Error().
				Err(err).
				Str("User", user.Uid).
				Str("Content", string(jsonData)).
				Msg("Invalid Credentials")
			c.JSON(http.StatusExpectationFailed, RequestErrorResult{
				errorCode: InvalidCredsError,
				content:   "Expected valid credentials for user",
			})
		}
	}
	content, err := sess.Message(request.Chat)
	if err != nil {
		sess.userMu.RLock()
		log.Error().
			Err(err).
			Str("User", sess.user.Uid).
			Msg("Invalid Credentials")
		sess.userMu.RUnlock()
		c.JSON(http.StatusExpectationFailed, RequestErrorResult{
			errorCode: InvalidCredsError,
			content:   "Expected valid credentials for user",
		})
	}
	c.String(http.StatusOK, content)
}

type WebsiteCreateUserRequest struct {
	Uid string `json:"uid"`
}

// query is the primary function used by ChatGPT to query data from this plugin, /
func initEmptyUserEndpoint(c *gin.Context) {
	var request WebsiteCreateUserRequest
	log.Debug().
		Msg("Creating new empty user")
	if err := c.BindJSON(&request); err != nil {
		log.Error().
			Err(err).
			Msg("Invalid json data on create user endpoint")
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: InvalidRequestContent,
			content:   "Content doesn't match expected structure",
		})
		return
	}
	if request.Uid == "" {
		log.Debug().Msg("empty UID")
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: InvalidRequestContent,
			content:   "Empty UID",
		})
		return
	}

	log.Debug().
		Str("user", request.Uid).
		Msg("Validating User")

	//validate UID as an account
	if !validateUIDBool(request.Uid) {
		return
	}

	user := User{}
	user.Uid = request.Uid

	// upload to firestore (user object) only if it doesn't exist,
	_, _, err := firestoreClient.Collection("users").Add(context.Background(), user)
	_, _, err = firestoreClient.Collection("users").Add(context.Background(), map[string]string{"yo": "dog"})
	if err != nil {
		log.Error().
			Err(err).
			Str("User", user.Uid).
			Msg("Failed to upload content to firebase")

		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: FirestoreError,
			content:   "Unable to upload document to firestore",
		})
		return
	}
	log.Debug().
		Str("User", user.Uid).
		Msg("Created new empty user")
	c.Status(http.StatusCreated)
}

type GetUserRequest struct {
	Uid string `json:"uid"`
}

func getUserEndpoint(c *gin.Context) {
	var request GetUserRequest
	if err := c.BindJSON(&request); err != nil {
		log.Error().
			Err(err).
			Msg("Invalid json data on get user endpoint")
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

	// validate UID exists
	//	if !validateUID(request.Uid, c) {
	//		return
	//	}

	docs, err := firestoreClient.Collection("users").Where("Uid", "==", request.Uid).Limit(1).Documents(context.Background()).GetAll()
	if err != nil || len(docs) == 0 {
		log.Debug().Str("User", request.Uid).Msg("Request trying to find invalid user")
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

func updateUserEndpoint(c *gin.Context) {
	log.Debug().Msg("Update User Endpoint Runnign")
	var request User
	if err := c.BindJSON(&request); err != nil {
		log.Error().
			Err(err).
			Msg("Invalid json data on get user endpoint")
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

	// validate UID exists
	// TODO: Uncomment
	/*	if !validateUID(request.Uid, c) {
		return
	}*/

	log.Debug().
		Str("User", request.Uid).
		Msg("getting user session")

	// upload the info for the current session
	if ses := GetSessionIfExists(request.Uid); ses != nil {
		ses.userMu.Lock()
		ses.user = request
		ses.userMu.Unlock()
	}

	log.Debug().
		Str("User", request.Uid).
		Msg("Updating Document")
	// Update Document
	docs, err := firestoreClient.Collection("users").Where("Uid", "==", request.Uid).Limit(1).Documents(context.Background()).GetAll()
	if err != nil || len(docs) == 0 {
		if err != nil {
			log.Error().
				Err(err).
				Msg("unable to find user")
		}

		log.Debug().Str("User", request.Uid).Msg("Request trying to find invalid user")
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: FirestoreError,
			content:   "Unable to find user in firestore",
		})
		return
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
