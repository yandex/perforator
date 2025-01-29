package service

import (
	"context"

	"github.com/yandex/perforator/perforator/proto/perforator"
)

////////////////////////////////////////////////////////////////////////////////

// ListServices implements perforator.PerforatorServer.
func (s *WebService) ListServices(ctx context.Context, req *perforator.ListServicesRequest) (*perforator.ListServicesResponse, error) {
	return s.client.ListServices(ctx, req)
}

// ListSuggestions implements perforator.PerforatorServer.
func (s *WebService) ListSuggestions(
	ctx context.Context,
	req *perforator.ListSuggestionsRequest,
) (*perforator.ListSuggestionsResponse, error) {
	return s.client.ListSuggestions(ctx, req)
}

// ListProfiles implements perforator.PerforatorServer.
func (s *WebService) ListProfiles(ctx context.Context, req *perforator.ListProfilesRequest) (*perforator.ListProfilesResponse, error) {
	return s.client.ListProfiles(ctx, req)
}

// GetProfile implements perforator.PerforatorServer.
func (s *WebService) GetProfile(ctx context.Context, req *perforator.GetProfileRequest) (*perforator.GetProfileResponse, error) {
	return s.client.GetProfile(ctx, req)
}

// MergeProfiles implements perforator.PerforatorServer.
func (s *WebService) MergeProfiles(ctx context.Context, req *perforator.MergeProfilesRequest) (*perforator.MergeProfilesResponse, error) {
	return s.client.MergeProfiles(ctx, req)
}

// UploadProfile implements perforator.PerforatorServer.
func (s *WebService) UploadProfile(ctx context.Context, req *perforator.UploadProfileRequest) (*perforator.UploadProfileResponse, error) {
	return s.client.UploadProfile(ctx, req)
}

////////////////////////////////////////////////////////////////////////////////

// ListMicroscopes implements perforator.MicroscopeService.
func (s *WebService) ListMicroscopes(ctx context.Context, req *perforator.ListMicroscopesRequest) (*perforator.ListMicroscopesResponse, error) {
	return s.client.ListMicroscopes(ctx, req)
}

// SetMicroscope implements perforator.MicroscopeService.
func (s *WebService) SetMicroscope(ctx context.Context, req *perforator.SetMicroscopeRequest) (*perforator.SetMicroscopeResponse, error) {
	return s.client.SetMicroscope(ctx, req)
}

////////////////////////////////////////////////////////////////////////////////

// GetTask implements perforator.TaskServiceServer.
func (s *WebService) GetTask(ctx context.Context, req *perforator.GetTaskRequest) (*perforator.GetTaskResponse, error) {
	return s.client.GetTask(ctx, req)
}

// StartTask implements perforator.TaskServiceServer.
func (s *WebService) StartTask(ctx context.Context, req *perforator.StartTaskRequest) (*perforator.StartTaskResponse, error) {
	return s.client.StartTask(ctx, req)
}

// ListTasks implements perforator.TaskServiceServer.
func (s *WebService) ListTasks(ctx context.Context, req *perforator.ListTasksRequest) (*perforator.ListTasksResponse, error) {
	return s.client.ListTasks(ctx, req)
}

////////////////////////////////////////////////////////////////////////////////
