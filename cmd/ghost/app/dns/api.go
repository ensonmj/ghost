package dns

import (
	"log"
	"net/http"

	"github.com/gin-contrib/static"
	"gopkg.in/gin-contrib/cors.v1"
	"gopkg.in/gin-gonic/gin.v1"
)

// StartAPIServer launches the API server
func StartAPIServer(debug bool) {
	if !debug {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.Use(cors.Default())
	// static files hav higher priority over dynamic routes
	router.Use(static.Serve("/", static.LocalFile("./public", true)))

	router.GET("/blockcache", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, gin.H{"length": gBlockCache.Length(), "items": gBlockCache.Items()})
	})

	router.GET("/blockcache/exists/:key", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, gin.H{"exists": gBlockCache.Exists(c.Param("key"))})
	})

	router.GET("/blockcache/get/:key", func(c *gin.Context) {
		if ok, _ := gBlockCache.Get(c.Param("key")); !ok {
			c.IndentedJSON(http.StatusOK, gin.H{"error": c.Param("key") + " not found"})
		} else {
			c.IndentedJSON(http.StatusOK, gin.H{"success": ok})
		}
	})

	router.GET("/blockcache/length", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, gin.H{"length": gBlockCache.Length()})
	})

	router.GET("/blockcache/remove/:key", func(c *gin.Context) {
		// Removes from BlockCache only. If the domain has already been queried and placed into MemoryCache, will need to wait until item is expired.
		gBlockCache.Remove(c.Param("key"))
		c.IndentedJSON(http.StatusOK, gin.H{"success": true})
	})

	router.GET("/blockcache/set/:key", func(c *gin.Context) {
		// MemoryBlockCache Set() always returns nil, so ignoring response.
		_ = gBlockCache.Set(c.Param("key"), true)
		c.IndentedJSON(http.StatusOK, gin.H{"success": true})
	})

	router.GET("/questioncache", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, gin.H{"length": gQuestionCache.Length(), "items": gQuestionCache.Backend})
	})

	router.GET("/questioncache/length", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, gin.H{"length": gQuestionCache.Length()})
	})

	router.GET("/questioncache/clear", func(c *gin.Context) {
		gQuestionCache.Clear()
		c.IndentedJSON(http.StatusOK, gin.H{"success": true})
	})

	router.GET("/questioncache/client/:client", func(c *gin.Context) {
		var filteredCache []QuestionCacheEntry

		gQuestionCache.mu.RLock()
		for _, entry := range gQuestionCache.Backend {
			if entry.Remote == c.Param("client") {
				filteredCache = append(filteredCache, entry)
			}
		}
		gQuestionCache.mu.RUnlock()

		c.IndentedJSON(http.StatusOK, filteredCache)
	})

	go func() {
		log.Println("API server listening on ", gConfig.API)
		if err := router.Run(gConfig.API); err != nil {
			log.Println("router return err ", err)
		}
	}()
}
