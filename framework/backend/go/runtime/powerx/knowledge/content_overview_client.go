package knowledge

import (
	"context"
	fw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
)

func (c *Client) ListContentOverview(ctx context.Context, query fw.ContentOverviewQuery) (*fw.ContentOverviewResult, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}
	var out struct {
		Result *fw.ContentOverviewResult `json:"result"`
	}
	meta, err := c.invokeContract(ctx, fw.ContentOverviewCapability, struct {
		Operation string                  `json:"operation"`
		Query     fw.ContentOverviewQuery `json:"query"`
	}{"list_articles", query}, &out)
	if err != nil {
		return nil, mapHostError(err)
	}
	if out.Result == nil {
		return nil, fw.NewError(fw.CodeInvalidResponse, "knowledgeContent.invalidResponse")
	}
	if err := out.Result.Validate(query); err != nil {
		return nil, err
	}
	out.Result.RequestID = meta.RequestID
	out.Result.InvocationTraceID = meta.InvocationTraceID
	if out.Result.TraceID == "" {
		out.Result.TraceID = meta.TraceID
	}
	return out.Result, nil
}

var _ fw.ContentOverviewService = (*Client)(nil)
