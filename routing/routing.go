package routing

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

// Route calls gin's routing function based on method and also calls OPTION on the route
func Route(router *gin.Engine, method string, route string, handler gin.HandlerFunc) {
	router.OPTIONS(route, empty)
	switch method {
	case http.MethodGet:
		router.GET(route, handler)
		return
	case http.MethodPost:
		router.POST(route, handler)
		return
	case http.MethodOptions:
		router.OPTIONS(route, handler)
		return
	case http.MethodDelete:
		router.DELETE(route, handler)
	}
}

// empty is a gin handler that returns a status code 200
func empty(c *gin.Context) {
	c.Status(http.StatusOK)
}

// CORS is a function that handles the CORS system for browser app interactions, this is a middleware function
func GENERAL(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Max-Age", "86400")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	c.Writer.Header().Set("Access-Control-Allow-Headers", strings.Join(c.Request.Header["Access-Control-Request-Headers"][:], ", "))
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Next()
}

// CORS is a function that handles the CORS system for browser app interactions, this is a middleware function
func CORS(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	c.Writer.Header().Set("Access-Control-Max-Age", "86400")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	c.Writer.Header().Set("Access-Control-Allow-Headers", strings.Join(c.Request.Header["Access-Control-Request-Headers"][:], ", "))
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Next()
}

// CORS is a function that handles the CORS system for browser app interactions, this is a middleware function
func CORS_STREAM(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Access-Control-Max-Age", "86400")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	c.Writer.Header().Set("Access-Control-Allow-Headers", strings.Join(c.Request.Header["Access-Control-Request-Headers"][:], ", "))
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Next()
}

func EMPTY_HANDLER(c *gin.Context) {

}
