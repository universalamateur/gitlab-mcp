package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/fforster/gitlab-mcp/lib/mcpargs"
)

const defaultPipelinesLimit = 1000

// PipelinesTools provides pipeline management capabilities.
type PipelinesTools struct{}

// AddTo registers pipeline tools with the MCP server.
func (p *PipelinesTools) AddTo(srv ToolAdder) {
	srv.AddTools(
		p.ListProjectPipelines(),
		p.GetPipeline(),
		p.CreatePipeline(),
		p.RetryPipeline(),
		p.CancelPipeline(),
	)
}

// --- list_project_pipelines ---

type listProjectPipelinesArgs struct {
	ProjectID mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	Status    string     `mcp_desc:"Filter pipelines by status: created, waiting_for_resource, preparing, pending, running, success, failed, canceled, skipped, manual, scheduled"`
	Source    string     `mcp_desc:"Filter pipelines by source: push, web, trigger, schedule, api, external, pipeline, chat, webide, merge_request_event, external_pull_request_event, parent_pipeline, ondemand_dast_scan, ondemand_dast_validation"`
	Ref       string     `mcp_desc:"Filter pipelines by git ref (branch or tag name)"`
	SHA       string     `mcp_desc:"Filter pipelines by commit SHA"`
	Username  string     `mcp_desc:"Filter pipelines by the username of the user who triggered them"`
	OrderBy   string     `mcp_desc:"Sort pipelines by the selected field. Default is 'id'" mcp_enum:"id,status,ref,updated_at,user_id"`
	SortOrder string     `mcp_desc:"Sort order to use. Default is 'desc'" mcp_enum:"asc,desc"`
	Limit     int        `mcp_desc:"Maximum number of pipelines to return. Defaults to 1000."`
}

// ListProjectPipelines returns a tool for listing pipelines in a project.
func (p *PipelinesTools) ListProjectPipelines() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(p.listProjectPipelines),
		Tool: mcpargs.NewTool("list_project_pipelines", listProjectPipelinesArgs{},
			mcp.WithDescription("Get a list of pipelines for a project"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (p *PipelinesTools) listProjectPipelines(ctx context.Context, client *gitlab.Client, args listProjectPipelinesArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.ListProjectPipelinesOptions{
		ListOptions: gitlab.ListOptions{PerPage: maxPerPage},
	}

	if args.Status != "" {
		opt.Status = gitlab.Ptr(gitlab.BuildStateValue(args.Status))
	}

	if args.Source != "" {
		opt.Source = gitlab.Ptr(args.Source)
	}

	if args.Ref != "" {
		opt.Ref = gitlab.Ptr(args.Ref)
	}

	if args.SHA != "" {
		opt.SHA = gitlab.Ptr(args.SHA)
	}

	if args.Username != "" {
		opt.Username = gitlab.Ptr(args.Username)
	}

	if args.OrderBy != "" {
		opt.OrderBy = gitlab.Ptr(args.OrderBy)
	}

	if args.SortOrder != "" {
		opt.Sort = gitlab.Ptr(args.SortOrder)
	}

	if args.Limit <= 0 {
		args.Limit = defaultPipelinesLimit
	}

	var pipelines []*gitlab.PipelineInfo

	for {
		p, resp, err := client.Pipelines.ListProjectPipelines(args.ProjectID.Value(), opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("ListProjectPipelines(%q): %w", args.ProjectID.Value(), err)
		}

		pipelines = append(pipelines, p...)

		if resp.NextPage == 0 || len(pipelines) >= args.Limit {
			break
		}

		opt.Page = resp.NextPage
	}

	if len(pipelines) > args.Limit {
		pipelines = pipelines[:args.Limit]
	}

	return newToolResultJSON(pipelines)
}

// --- get_pipeline ---

type getPipelineArgs struct {
	ProjectID  mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	PipelineID int64      `mcp_desc:"ID of the pipeline" mcp_required:"true"`
}

// GetPipeline returns a tool for getting a single pipeline.
func (p *PipelinesTools) GetPipeline() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(p.getPipeline),
		Tool: mcpargs.NewTool("get_pipeline", getPipelineArgs{},
			mcp.WithDescription("Get a single pipeline by ID"),
			mcp.WithReadOnlyHintAnnotation(true),
		),
	}
}

func (p *PipelinesTools) getPipeline(ctx context.Context, client *gitlab.Client, args getPipelineArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pipeline, _, err := client.Pipelines.GetPipeline(args.ProjectID.Value(), args.PipelineID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("GetPipeline(%q, %d): %w", args.ProjectID.Value(), args.PipelineID, err)
	}

	return newToolResultJSON(pipeline)
}

// --- create_pipeline ---

type createPipelineArgs struct {
	ProjectID mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	Ref       string     `mcp_desc:"Branch or tag to run the pipeline for" mcp_required:"true"`
	Variables string     `mcp_desc:"Comma-separated list of variables in KEY=VALUE format (e.g. 'FOO=bar,BAZ=qux')"`
}

// CreatePipeline returns a tool for creating a new pipeline.
func (p *PipelinesTools) CreatePipeline() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(p.createPipeline),
		Tool: mcpargs.NewTool("create_pipeline", createPipelineArgs{},
			mcp.WithDescription("Create a new pipeline for a branch or tag, optionally with variables"),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
		),
	}
}

func (p *PipelinesTools) createPipeline(ctx context.Context, client *gitlab.Client, args createPipelineArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opt := &gitlab.CreatePipelineOptions{
		Ref: gitlab.Ptr(args.Ref),
	}

	if args.Variables != "" {
		vars, err := parsePipelineVariables(args.Variables)
		if err != nil {
			return nil, err
		}

		opt.Variables = vars
	}

	pipeline, _, err := client.Pipelines.CreatePipeline(args.ProjectID.Value(), opt, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("CreatePipeline(%q): %w", args.ProjectID.Value(), err)
	}

	return newToolResultJSON(pipeline)
}

// --- retry_pipeline ---

type retryPipelineArgs struct {
	ProjectID  mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	PipelineID int64      `mcp_desc:"ID of the pipeline" mcp_required:"true"`
}

// RetryPipeline returns a tool for retrying all failed jobs in a pipeline.
func (p *PipelinesTools) RetryPipeline() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(p.retryPipeline),
		Tool: mcpargs.NewTool("retry_pipeline", retryPipelineArgs{},
			mcp.WithDescription("Retry all failed jobs in a pipeline"),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
		),
	}
}

func (p *PipelinesTools) retryPipeline(ctx context.Context, client *gitlab.Client, args retryPipelineArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pipeline, _, err := client.Pipelines.RetryPipelineBuild(args.ProjectID.Value(), args.PipelineID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("RetryPipelineBuild(%q, %d): %w", args.ProjectID.Value(), args.PipelineID, err)
	}

	return newToolResultJSON(pipeline)
}

// --- cancel_pipeline ---

type cancelPipelineArgs struct {
	ProjectID  mcpargs.ID `mcp_desc:"ID of the project either in owner/project format or the numeric project ID" mcp_required:"true"`
	PipelineID int64      `mcp_desc:"ID of the pipeline" mcp_required:"true"`
}

// CancelPipeline returns a tool for canceling a running pipeline.
func (p *PipelinesTools) CancelPipeline() server.ServerTool {
	return server.ServerTool{
		Handler: ClientHandler(p.cancelPipeline),
		Tool: mcpargs.NewTool("cancel_pipeline", cancelPipelineArgs{},
			mcp.WithDescription("Cancel a running pipeline"),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
		),
	}
}

func (p *PipelinesTools) cancelPipeline(ctx context.Context, client *gitlab.Client, args cancelPipelineArgs, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pipeline, _, err := client.Pipelines.CancelPipelineBuild(args.ProjectID.Value(), args.PipelineID, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("CancelPipelineBuild(%q, %d): %w", args.ProjectID.Value(), args.PipelineID, err)
	}

	return newToolResultJSON(pipeline)
}

// --- helpers ---

// parsePipelineVariables parses a comma-separated "KEY=VALUE" string into pipeline variables.
func parsePipelineVariables(s string) (*[]*gitlab.PipelineVariableOptions, error) {
	var vars []*gitlab.PipelineVariableOptions

	for part := range strings.SplitSeq(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		key, value, found := strings.Cut(part, "=")
		if !found {
			return nil, fmt.Errorf("%w: variable %q must be in KEY=VALUE format", ErrInvalidArgument, part)
		}

		vars = append(vars, &gitlab.PipelineVariableOptions{
			Key:   gitlab.Ptr(key),
			Value: gitlab.Ptr(value),
		})
	}

	return &vars, nil
}
