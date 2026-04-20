package test

import (
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	pb "github.com/rozoomcool/sihkaromicro/projects/gen/proto/projects"
	"github.com/rozoomcool/sihkaromicro/projects/internal/dto"
	"github.com/rozoomcool/sihkaromicro/projects/internal/handler"
	"github.com/rozoomcool/sihkaromicro/projects/internal/model"
	svcmock "github.com/rozoomcool/sihkaromicro/projects/internal/service/mock"
	"github.com/rozoomcool/sihkaromicro/projects/internal/testutil"
)

var testLogger = slog.New(slog.NewTextHandler(os.Stdout, nil))

func TestCreateProject(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		req       *pb.CreateProjectRequest
		mockSetup func(*svcmock.MockProjectService)
		wantErr   bool
	}{
		{
			name:   "success",
			userID: "user-123",
			req:    &pb.CreateProjectRequest{Title: "My Project"},
			mockSetup: func(m *svcmock.MockProjectService) {
				m.On("Create", mock.Anything, "user-123", "My Project").
					Return(&model.Project{Base: model.Base{ID: 34}, OwnerID: "user-123", Title: "My Project"}, nil)
			},
			wantErr: false,
		},
		{
			name:   "empty title",
			userID: "user-123",
			req:    &pb.CreateProjectRequest{Title: ""},
			mockSetup: func(m *svcmock.MockProjectService) {
				m.On("Create", mock.Anything, "user-123", "").
					Return((*model.Project)(nil), errors.New("title is required"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := new(svcmock.MockProjectService)
			tt.mockSetup(svc)

			h := handler.NewProjectGRPCHandler(svc, testLogger)
			ctx := testutil.CtxWithUserID(tt.userID)

			resp, err := h.CreateProject(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.req.Title, resp.Project.Title)
			}

			svc.AssertExpectations(t)
		})
	}
}

func TestListProjects(t *testing.T) {
	svc := new(svcmock.MockProjectService)
	svc.On("List", mock.Anything, "user-123", 1, 10).
		Return(&dto.PageWrapper[model.Project]{
			Total: 2,
			Data: []model.Project{
				{Base: model.Base{ID: 1}, OwnerID: "user-123", Title: "Project A"},
				{Base: model.Base{ID: 2}, OwnerID: "user-123", Title: "Project B"},
			},
		}, nil)

	h := handler.NewProjectGRPCHandler(svc, testLogger)
	ctx := testutil.CtxWithUserID("user-123")

	resp, err := h.ListProjects(ctx, &pb.ListProjectsRequest{Page: 1, PageSize: 10})

	assert.NoError(t, err)
	assert.Equal(t, int32(2), resp.Total)
	assert.Len(t, resp.Projects, 2)

	svc.AssertExpectations(t)
}
