package localagent

import (
	"encoding/json"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	agentfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/agent"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	skillfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/skills"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	svc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/localagent"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// This plugin-owned management surface has no PowerX dependency or mode override.
func RegisterRoutes(admin *gin.RouterGroup, deps *app.Deps) {
	if deps == nil || deps.DB == nil {
		return
	}
	var ai svc.AI
	if deps.LocalAI != nil {
		ai = deps.LocalAI
	}
	service := svc.New(deps.DB, ai, deps.LocalSkillInvoker)
	agents, _ := agentfw.NewManagementRuntime(provider.ModeLocal, service, nil)
	skills, _ := skillfw.NewManagementRuntime(provider.ModeLocal, service, nil)
	a, _ := agents.Resolve()
	s, _ := skills.Resolve()
	g := admin.Group("/local-intelligence", func(c *gin.Context) {
		if tc, ok := authx.GetTenantContext(c); ok {
			c.Request = c.Request.WithContext(svc.WithActor(c.Request.Context(), tc.MemberUUID))
		}
		c.Next()
	})
	g.GET("/models", func(c *gin.Context) { out, e := service.Models(c.Request.Context()); respond(c, out, e) })
	g.GET("/agents", func(c *gin.Context) { out, e := a.ListAgents(c.Request.Context(), query(c)); respond(c, out, e) })
	g.GET("/skills", func(c *gin.Context) { out, e := s.ListSkills(c.Request.Context(), query(c)); respond(c, out, e) })
	saveAgent := func(c *gin.Context) {
		var in agentfw.DefinitionInput
		if !bind(c, &in) {
			return
		}
		out, e := a.SaveAgent(c.Request.Context(), c.Param("uuid"), in)
		respond(c, out, e)
	}
	saveSkill := func(c *gin.Context) {
		var in skillfw.DefinitionInput
		if !bind(c, &in) {
			return
		}
		out, e := s.SaveSkill(c.Request.Context(), c.Param("uuid"), in)
		respond(c, out, e)
	}
	g.POST("/agents", saveAgent)
	g.PUT("/agents/:uuid", saveAgent)
	g.POST("/skills", saveSkill)
	g.PUT("/skills/:uuid", saveSkill)
	g.POST("/agents/:uuid/debug", func(c *gin.Context) {
		var in agentfw.DebugInput
		if !bind(c, &in) {
			return
		}
		out, e := a.DebugAgent(c.Request.Context(), c.Param("uuid"), in)
		respond(c, out, e)
	})
	g.POST("/skills/:uuid/invoke", func(c *gin.Context) {
		var in struct {
			Input map[string]any `json:"input"`
		}
		if !bind(c, &in) {
			return
		}
		out, e := s.InvokeSkill(c.Request.Context(), c.Param("uuid"), in.Input)
		respond(c, out, e)
	})
}
func query(c *gin.Context) skillfw.DefinitionQuery {
	p, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("page_size"))
	return skillfw.DefinitionQuery{Query: c.Query("q"), Status: c.Query("status"), Page: p, PageSize: size}
}
func bind(c *gin.Context, out any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 128<<10)
	d := json.NewDecoder(c.Request.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(out); e != nil {
		contracts.ResponseError(c, 400, "LOCAL_AGENT_INVALID_ARGUMENT", "LOCAL_AGENT_INVALID_ARGUMENT")
		return false
	}
	var trailing any
	if e := d.Decode(&trailing); e != io.EOF {
		contracts.ResponseError(c, 400, "LOCAL_AGENT_INVALID_ARGUMENT", "LOCAL_AGENT_INVALID_ARGUMENT")
		return false
	}
	return true
}
func respond(c *gin.Context, out any, err error) {
	if err != nil {
		status, code := svc.ErrorStatus(err)
		slog.WarnContext(c.Request.Context(), "local_intelligence.operation_failed", "code", code, "status", status, "route", c.FullPath(), "error", err)
		contracts.ResponseError(c, status, code, code)
		return
	}
	contracts.ResponseSuccess(c, out)
}
