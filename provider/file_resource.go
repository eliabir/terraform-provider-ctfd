package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ctfer-io/go-ctfd/api"
	"github.com/eliabir/terraform-provider-ctfd/v2/provider/utils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                 = (*fileResource)(nil)
	_ resource.ResourceWithConfigure    = (*fileResource)(nil)
	_ resource.ResourceWithImportState  = (*fileResource)(nil)
	_ resource.ResourceWithUpgradeState = (*fileResource)(nil)
	_ resource.ResourceWithModifyPlan   = (*fileResource)(nil)
)

func NewFileResource() resource.Resource {
	return &fileResource{}
}

type fileResource struct {
	fm *Framework
}

type fileResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ChallengeID types.String `tfsdk:"challenge_id"`
	Name        types.String `tfsdk:"name"`
	Location    types.String `tfsdk:"location"`
	SHA1Sum     types.String `tfsdk:"sha1sum"`
	ContentPath types.String `tfsdk:"content_path"`
	ContentHash types.String `tfsdk:"content_hash"`
}

func (r *fileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file"
}

func (r *fileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: "A CTFd file for a challenge.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of the file, used internally to handle the CTFd corresponding object. WARNING: updating this file does not work, requires full replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"challenge_id": schema.StringAttribute{
				MarkdownDescription: "Challenge of the file.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the file as displayed to end-users.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"location": schema.StringAttribute{
				MarkdownDescription: "Location where the file is stored on the CTFd instance, for download purposes.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"sha1sum": schema.StringAttribute{
				MarkdownDescription: "The sha1 sum of the file.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"content_path": schema.StringAttribute{
				MarkdownDescription: "Local filesystem path to the file's content. Provide it with `\"${path.module}/...\"`.",
				Required:            true,
			},
			"content_hash": schema.StringAttribute{
				MarkdownDescription: "Hash of the file content, used to detect changes since the content itself is not stored in state. Set it to `filesha256(\"${path.module}/...\")`.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *fileResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id":           schema.StringAttribute{},
					"challenge_id": schema.StringAttribute{},
					"name":         schema.StringAttribute{},
					"location":     schema.StringAttribute{},
					"sha1sum":      schema.StringAttribute{},
					"contentb64":   schema.StringAttribute{},
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				type priorModel struct {
					ID          types.String `tfsdk:"id"`
					ChallengeID types.String `tfsdk:"challenge_id"`
					Name        types.String `tfsdk:"name"`
					Location    types.String `tfsdk:"location"`
					SHA1Sum     types.String `tfsdk:"sha1sum"`
					ContentB64  types.String `tfsdk:"contentb64"`
				}

				var prior priorModel
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgraded := fileResourceModel{
					ID:          prior.ID,
					ChallengeID: prior.ChallengeID,
					Name:        prior.Name,
					Location:    prior.Location,
					SHA1Sum:     prior.SHA1Sum,
					ContentPath: types.StringNull(),
					ContentHash: types.StringNull(),
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
			},
		},
	}
}

func (r *fileResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Only relevant when updating an existing resource (both state and plan set).
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var state, plan fileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// content_path is only used to upload the bytes on create/replace. Its value
	// varies by machine (e.g. CI vs local absolute paths), so ignore drift on it
	// while the resource stays in place. If a RequiresReplace attribute changed
	// (name/filename, challenge_id or content_hash), the resource is recreated and
	// content_path must keep its configured value so the new content is uploaded.
	if plan.Name.Equal(state.Name) &&
		plan.ChallengeID.Equal(state.ChallengeID) &&
		plan.ContentHash.Equal(state.ContentHash) {
		plan.ContentPath = state.ContentPath
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

func (r *fileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	fm, ok := req.ProviderData.(*Framework)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected %T, got: %T. Please open an issue at https://github.com/eliabir/terraform-provider-ctfd", (*Framework)(nil), req.ProviderData),
		)
		return
	}

	r.fm = fm
}

func (r *fileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	ctx, span := StartTFSpan(ctx, r.fm.Tp.Tracer(serviceName), r)
	defer span.End()

	var data fileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	content, err := os.ReadFile(data.ContentPath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Content Error",
			fmt.Sprintf("unable to read file at %q: %s", data.ContentPath.ValueString(), err),
		)
		return
	}
	params := &api.PostFilesParams{
		Files: []*api.InputFile{
			{
				Name:    data.Name.ValueString(),
				Content: content,
			},
		},
		Location: data.Location.ValueStringPointer(),
	}
	if !data.ChallengeID.IsNull() {
		params.Challenge = utils.Ptr(utils.Atoi(data.ChallengeID.ValueString()))
	}
	res, _, err := r.fm.Client.PostFiles(ctx, params, WithTracerProvider(r.fm.Tp))
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create file, got error: %s", err),
		)
		return
	}

	tflog.Trace(ctx, "created a file")

	// Save computed attributes in state
	data.ID = types.StringValue(strconv.Itoa(res[0].ID))
	data.SHA1Sum = types.StringValue(res[0].SHA1sum)
	data.Location = types.StringValue(res[0].Location)

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *fileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx, span := StartTFSpan(ctx, r.fm.Tp.Tracer(serviceName), r)
	defer span.End()

	var data fileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	res, _, err := r.fm.Client.GetFile(ctx, data.ID.ValueString(), WithTracerProvider(r.fm.Tp))
	if err != nil {
		resp.Diagnostics.AddError(
			"CTFd Error",
			fmt.Sprintf("Unable to retrieve file %s, got error: %s", data.ID.ValueString(), err),
		)
		return
	}

	// Preserve the user-provided name. CTFd sanitises the stored filename in the
	// location (e.g. spaces become underscores), so deriving name from it would
	// cause perpetual drift/replacement. Only fall back to the location basename
	// on import, when state has no name yet.
	if data.Name.IsNull() || data.Name.ValueString() == "" {
		data.Name = types.StringValue(filepath.Base(res.Location))
	}
	data.Location = types.StringValue(res.Location)
	data.SHA1Sum = types.StringValue(res.SHA1sum)
	data.ChallengeID = lookForChallengeId(ctx, r.fm.Client, res.ID, resp.Diagnostics, WithTracerProvider(r.fm.Tp))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *fileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	ctx, span := StartTFSpan(ctx, r.fm.Tp.Tracer(serviceName), r)
	defer span.End()

	var data fileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddError("Provider Error", "CTFd does not permit update of file-related information thus this provider cannot do so. This operation should not have been possible.")
}

func (r *fileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	ctx, span := StartTFSpan(ctx, r.fm.Tp.Tracer(serviceName), r)
	defer span.End()

	var data fileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.fm.Client.DeleteFile(ctx, data.ID.ValueString(), WithTracerProvider(r.fm.Tp)); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete file %s, got error: %s", data.ID.ValueString(), err))
		return
	}
}

func (r *fileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	// Automatically call r.Read
}

// XXX this helper only exist because CTFd does not return the challenge id of a file if it exist...
func lookForChallengeId(ctx context.Context, client *Client, fileID int, diags diag.Diagnostics, opts ...Option) types.String {
	challs, _, err := client.GetChallenges(ctx, &api.GetChallengesParams{
		View: utils.Ptr("admin"), // required, else CTFd only returns the "visible" challenges
	}, opts...)
	if err != nil {
		diags.AddError(
			"CTFd Error",
			fmt.Sprintf("Unable to query challenges, got error: %s", err),
		)
		return types.StringNull()
	}

	for _, chall := range challs {
		files, _, err := client.GetChallengeFiles(ctx, strconv.Itoa(chall.ID), opts...)
		if err != nil {
			diags.AddError(
				"CTFd Error",
				fmt.Sprintf("Unable to query challenge %d files, got error: %s", chall.ID, err),
			)
			return types.StringNull()
		}
		for _, file := range files {
			if file.ID == fileID {
				return types.StringValue(strconv.Itoa(chall.ID))
			}
		}
	}
	return types.StringNull()
}
