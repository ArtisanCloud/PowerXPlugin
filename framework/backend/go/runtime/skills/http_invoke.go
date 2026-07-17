package skills

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (a *HTTPAdapter) Invoke(c *gin.Context) {
	var inv PluginSkillInvocation
	if err := c.ShouldBindJSON(&inv); err != nil {
		status, result := MapError(NewError(ErrCodeInvalidInvocation, err.Error()), inv)
		c.JSON(status, result)
		return
	}
	result, err := a.registry.Invoke(c.Request.Context(), inv)
	if err != nil {
		status, mapped := MapError(err, inv)
		c.JSON(status, mapped)
		return
	}
	status := http.StatusOK
	if result.Status == ResultQueued || result.Status == ResultRunning {
		status = http.StatusAccepted
	}
	c.JSON(status, result)
}
