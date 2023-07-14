package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
)

type RequestErrorResult struct {
	errorCode WebsiteRequestError `json:"error_code"`
	content   string              `json:"content"`
}

type WebsiteCreateUserRequest struct {
	uid string `json:"uid"`
}

func validateUID(uid string, c *gin.Context) bool {
	authMutex.Lock()
	_, err := fireauthClient.GetUser(context.Background(), uid)
	authMutex.Unlock()
	if err != nil {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: NonExistentUser,
			content:   "This user does not exist, invalid UID",
		})
		return false
	}
	return true
}

// query is the primary function used by ChatGPT to query data from this plugin, /
func initEmptyUserEndpoint(c *gin.Context) {
	var request WebsiteCreateUserRequest
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: InvalidRequestContent,
			content:   "Content doesn't match expected structure",
		})
		return
	}
	if request.uid == "" {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: InvalidRequestContent,
			content:   "Empty UID",
		})
		return
	}

	//validate UID
	if !validateUID(request.uid, c) {
		return
	}

	data, err := UserWithToken()
	if err != nil {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: ServerError,
			content:   "Unable to generate a new user with a valid token",
		})
		return
	}

	data.Uid = request.uid
	// upload to firestore (user object) only if it doesn't exist,
	firestoreMutex.Lock()
	_, _, err = firestoreClient.Collection("users").Add(context.Background(), data)
	firestoreMutex.Unlock()
	if err != nil {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: FirestoreError,
			content:   "Unable to upload document to firestore",
		})
		return
	}
	c.Status(http.StatusCreated)
}

type GetUserRequest struct {
	uid string `json:"uid"`
}

func getUserEndpoint(c *gin.Context) {
	var request GetUserRequest
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: InvalidRequestContent,
			content:   "Content doesn't match expected structure",
		})
		return
	}
	if request.uid == "" {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: InvalidRequestContent,
			content:   "Empty UID",
		})
		return
	}

	// validate UID exists
	if !validateUID(request.uid, c) {
		return
	}

	firestoreMutex.Lock()
	docs, err := firestoreClient.Collection("users").Where("uid", "==", request.uid).Limit(1).Documents(context.Background()).GetAll()
	firestoreMutex.Unlock()
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

func updateUserEndpoint(c *gin.Context) {
	// read request data and uid

	var request User
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

	// validate UID exists
	if !validateUID(request.Uid, c) {
		return
	}

	// Update Document
	firestoreMutex.Lock()
	docs, err := firestoreClient.Collection("users").Where("uid", "==", request.Uid).Limit(1).Documents(context.Background()).GetAll()
	firestoreMutex.Unlock()
	if err != nil || len(docs) == 0 {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: FirestoreError,
			content:   "Unable to find user in firestore",
		})
		return
	}

	doc := docs[0]
	_, err = doc.Ref.Set(context.Background(), request)
	if err != nil {
		c.JSON(http.StatusBadRequest, RequestErrorResult{
			errorCode: FirestoreError,
			content:   "Unable to Update File in Firestore",
		})
		return
	}
	c.Status(http.StatusOK)
}
