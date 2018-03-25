package main

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	router.GET("/packages/:pkg", func(c *gin.Context) {
		c.File("/hexdump/packages/" + c.Param("pkg"))
	})

	router.GET("/tarballs/:tarfile", func(c *gin.Context) {
		c.File("/hexdump/tarballs/" + c.Param("tarfile"))
	})

	router.GET("/installs/*file", func(c *gin.Context) {
		fmt.Println("In here...")
		path := strings.Split(c.Param("file"), "/")
		fmt.Println(path)
		switch len(path) {
		case 2:
			c.File("/hexdump/installs/" + path[1])
		case 3:
			if strings.HasSuffix(path[2], ".ez") {
				c.File("/hexdump/installs/" + path[1] + "-" + path[2])
			} else {
				c.File("/hexdump/installs/" + path[1] + "-" + path[2] + ".ez")
			}
		}
	})

	router.StaticFile("/registry.ets.gz", "/hexdump/registry.ets.gz")

	// Listen and serve on 0.0.0.0:5000
	router.Run(":5000")
}
