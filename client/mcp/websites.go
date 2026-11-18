package mcp

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	listWebsitesToolName         = "list_websites"
	websiteInfoToolName          = "website_info"
	websiteRemoveToolName        = "website_remove"
	websiteAddContentToolName    = "website_add_content"
	websiteUpdateContentToolName = "website_update_content"
	websiteRemoveContentToolName = "website_remove_content"
)

// --- Args ---

type websiteInfoArgs struct {
	Name string `json:"name" jsonschema:"description=Website name,required"`
}

type websiteRemoveArgs struct {
	Name string `json:"name" jsonschema:"description=Website name to remove,required"`
}

type websiteAddContentArgs struct {
	WebsiteName string `json:"website_name" jsonschema:"description=Website name,required"`
	Path        string `json:"path" jsonschema:"description=URL path for the content,required"`
	ContentType string `json:"content_type" jsonschema:"description=MIME content type,required"`
	Content     string `json:"content" jsonschema:"description=Base64-encoded content,required"`
}

type websiteUpdateContentArgs struct {
	WebsiteName string `json:"website_name" jsonschema:"description=Website name,required"`
	Path        string `json:"path" jsonschema:"description=URL path to update,required"`
	ContentType string `json:"content_type" jsonschema:"description=MIME content type,required"`
	Content     string `json:"content" jsonschema:"description=Base64-encoded content,required"`
}

type websiteRemoveContentArgs struct {
	Name  string   `json:"name" jsonschema:"description=Website name,required"`
	Paths []string `json:"paths" jsonschema:"description=URL paths to remove,required"`
}

// --- Results ---

type webContentItem struct {
	Path        string `json:"path"`
	ContentType string `json:"content_type"`
	Size        uint64 `json:"size"`
}

type websiteItem struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Contents []webContentItem `json:"contents,omitempty"`
}

type websiteListResult struct {
	Websites []websiteItem `json:"websites"`
	Count    int           `json:"count"`
}

// --- Handlers ---

func (s *SliverMCPServer) listWebsitesHandler(ctx context.Context, _ mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}
	s.logToolCall(listWebsitesToolName, "", "")

	resp, err := s.Rpc.Websites(ctx, &commonpb.Empty{})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("list_websites failed", err), nil
	}

	result := websiteListResult{Websites: make([]websiteItem, 0, len(resp.Websites))}
	for _, w := range resp.Websites {
		if w == nil {
			continue
		}
		result.Websites = append(result.Websites, websiteItem{
			ID:   w.ID,
			Name: w.Name,
		})
	}
	result.Count = len(result.Websites)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) websiteInfoHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args websiteInfoArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(websiteInfoToolName, "", "")

	resp, err := s.Rpc.Website(ctx, &clientpb.Website{Name: args.Name})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("website_info failed", err), nil
	}

	result := websiteItem{
		ID:       resp.ID,
		Name:     resp.Name,
		Contents: make([]webContentItem, 0),
	}
	for path, wc := range resp.Contents {
		if wc == nil {
			continue
		}
		result.Contents = append(result.Contents, webContentItem{
			Path:        path,
			ContentType: wc.ContentType,
			Size:        wc.Size,
		})
	}

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) websiteRemoveHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args websiteRemoveArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(websiteRemoveToolName, "", "")

	if _, err := s.Rpc.WebsiteRemove(ctx, &clientpb.Website{Name: args.Name}); err != nil {
		return mcpapi.NewToolResultErrorFromErr("website_remove failed", err), nil
	}

	type rmResult struct {
		Name    string `json:"name"`
		Removed bool   `json:"removed"`
	}
	return mcpapi.NewToolResultStructuredOnly(rmResult{Name: args.Name, Removed: true}), nil
}

func (s *SliverMCPServer) websiteAddContentHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args websiteAddContentArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(websiteAddContentToolName, "", "")

	data, err := base64.StdEncoding.DecodeString(args.Content)
	if err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid base64 content: %v", err)), nil
	}

	resp, err := s.Rpc.WebsiteAddContent(ctx, &clientpb.WebsiteAddContent{
		Name: args.WebsiteName,
		Contents: map[string]*clientpb.WebContent{
			args.Path: {
				ContentType: args.ContentType,
				Content:     data,
			},
		},
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("website_add_content failed", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(websiteItem{
		ID:   resp.ID,
		Name: resp.Name,
	}), nil
}

func (s *SliverMCPServer) websiteUpdateContentHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args websiteUpdateContentArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(websiteUpdateContentToolName, "", "")

	data, err := base64.StdEncoding.DecodeString(args.Content)
	if err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid base64 content: %v", err)), nil
	}

	resp, err := s.Rpc.WebsiteUpdateContent(ctx, &clientpb.WebsiteAddContent{
		Name: args.WebsiteName,
		Contents: map[string]*clientpb.WebContent{
			args.Path: {
				ContentType: args.ContentType,
				Content:     data,
			},
		},
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("website_update_content failed", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(websiteItem{
		ID:   resp.ID,
		Name: resp.Name,
	}), nil
}

func (s *SliverMCPServer) websiteRemoveContentHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args websiteRemoveContentArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(websiteRemoveContentToolName, "", "")

	resp, err := s.Rpc.WebsiteRemoveContent(ctx, &clientpb.WebsiteRemoveContent{
		Name:  args.Name,
		Paths: args.Paths,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("website_remove_content failed", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(websiteItem{
		ID:   resp.ID,
		Name: resp.Name,
	}), nil
}
