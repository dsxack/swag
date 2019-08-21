package main

import (
	"github.com/dsxack/swag/testdata/struct_comment/api"
	"github.com/gin-gonic/gin"
)

// @title Swagger Example API
// @version 1.0
// @description This is a sample server Petstore server.
// @host localhost:4000
// @basePath /api
func main() {
	r := gin.New()
	r.GET("/posts/:post_id", api.GetPost)
	r.Run()
}
