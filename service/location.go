package service

import (
	"github.com/gin-gonic/gin"
	"net/url"
)

const (
	key = "location"
)

// Get returns the Location information for the incoming http.Request from the
// context. If the location is not set a nil value is returned.
func Get(c *gin.Context) *url.URL {
	v, ok := c.Get(key)

	if !ok {
		return nil
	}

	vv, ok := v.(*url.URL)

	if !ok {
		return nil
	}

	return vv
}
