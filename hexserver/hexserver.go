package main

import "github.com/gin-gonic/gin"

func main() {
	router := gin.Default()

	router.GET("/packages/:pkg", func(c *gin.Context) {
		c.File("/hexdump/packages/" + c.Param("pkg"))
	})

	router.GET("/tarballs/:tarfile", func(c *gin.Context) {
		c.File("/hexdump/tarballs/" + c.Param("tarfile"))
	})

	router.StaticFile("/registry.ets.gz", "/hexdump/registry.ets.gz")

	// Listen and serve on 0.0.0.0:5000
	router.Run(":5000")
}
