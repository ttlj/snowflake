package restful

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ttlj/snowflake"
)

// Env contains snowflake instance
type Env struct {
	Flake *snowflake.Node
}

// NewEngine creates GIN server
func NewEngine(e *Env) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	//r.Use(gin.Logger())
	r.GET("/status", statusHandler)
	r.GET("/id", e.idHandler)
	r.GET("/ids", e.idsHandler)
	return r
}

func (e *Env) idHandler(c *gin.Context) {
	// Generate new ID
	id, err := e.Flake.NextID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	} else {
		// Return ID as string
		c.JSON(http.StatusOK, gin.H{
			"id": fmt.Sprint(id),
		})
	}
}

func (e *Env) idsHandler(c *gin.Context) {
	lst, err := e.Flake.NextIDs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"result": "Failed to generate unique integer id list"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ids": fmt.Sprint(lst),
	})
}

func statusHandler(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}
