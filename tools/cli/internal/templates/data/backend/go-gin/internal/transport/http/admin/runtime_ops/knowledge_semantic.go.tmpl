package runtime_ops

import (
	"encoding/json"
	"errors"
	fw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
)

func knowledgeRequestID(err error) string {
	var target *fw.Error
	if errors.As(err, &target) {
		return target.RequestID
	}
	return ""
}
func semanticBody(c *gin.Context, out any) error {
	if c.Query("tenant_uuid") != "" {
		return fw.NewError(fw.CodeInvalidArgument, "knowledge.semantic.invalid_argument")
	}
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 128<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return fw.NewError(fw.CodeInvalidArgument, "knowledge.semantic.invalid_argument")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return fw.NewError(fw.CodeInvalidArgument, "knowledge.semantic.invalid_argument")
	}
	return nil
}
func (h *KnowledgeHandler) semanticService(c *gin.Context) (fw.SemanticKnowledgeProvider, error) {
	if c.Query("tenant_uuid") != "" {
		return nil, fw.NewError(fw.CodeInvalidArgument, "knowledge.semantic.invalid_argument")
	}
	provider, err := h.provider()
	if err != nil {
		return nil, err
	}
	return fw.SemanticProvider(provider)
}
func (h *KnowledgeHandler) LabSemanticCapabilities(c *gin.Context) {
	service, err := h.semanticService(c)
	if err != nil {
		writeLabKnowledgeError(c, err)
		return
	}
	item, err := service.GetSemanticCapabilities(c.Request.Context(), c.Param("spaceID"))
	if err != nil {
		writeLabKnowledgeError(c, err)
		return
	}
	c.JSON(200, gin.H{"success": true, "data": gin.H{"capabilities": item}})
}
func (h *KnowledgeHandler) LabConfigureSemantic(c *gin.Context) {
	var input fw.SemanticConfigureInput
	if err := semanticBody(c, &input); err != nil {
		writeLabKnowledgeError(c, err)
		return
	}
	service, err := h.semanticService(c)
	if err != nil {
		writeLabKnowledgeError(c, err)
		return
	}
	item, err := service.ConfigureSemanticIndex(c.Request.Context(), c.Param("spaceID"), input)
	if err != nil {
		writeLabKnowledgeError(c, err)
		return
	}
	c.JSON(200, gin.H{"success": true, "data": gin.H{"generation": item}})
}
func (h *KnowledgeHandler) LabSemanticQuery(c *gin.Context) {
	var input fw.SemanticQuery
	if err := semanticBody(c, &input); err != nil {
		writeLabKnowledgeError(c, err)
		return
	}
	service, err := h.semanticService(c)
	if err != nil {
		writeLabKnowledgeError(c, err)
		return
	}
	item, err := service.QuerySemanticKnowledge(c.Request.Context(), input)
	if err != nil {
		writeLabKnowledgeError(c, err)
		return
	}
	c.JSON(200, gin.H{"success": true, "data": gin.H{"result": item}})
}
func (h *KnowledgeHandler) LabSemanticVisibility(c *gin.Context) {
	var input fw.SemanticVisibilityInput
	if err := semanticBody(c, &input); err != nil {
		writeLabKnowledgeError(c, err)
		return
	}
	service, err := h.semanticService(c)
	if err != nil {
		writeLabKnowledgeError(c, err)
		return
	}
	item, err := service.SetDocumentVisibility(c.Request.Context(), c.Param("spaceID"), c.Param("documentID"), input)
	if err != nil {
		writeLabKnowledgeError(c, err)
		return
	}
	c.JSON(200, gin.H{"success": true, "data": gin.H{"item": item}})
}
