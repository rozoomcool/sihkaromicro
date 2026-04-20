package mock

import (
	"context"

	"github.com/rozoomcool/sihkaromicro/projects/internal/dto"
	"github.com/rozoomcool/sihkaromicro/projects/internal/model"
	"github.com/stretchr/testify/mock"
)

type MockProjectService struct {
	mock.Mock
}

func (m *MockProjectService) Create(ctx context.Context, ownerID, title string) (*model.Project, error) {
	args := m.Called(ctx, ownerID, title)
	return args.Get(0).(*model.Project), args.Error(1)
}

func (m *MockProjectService) Get(ctx context.Context, ownerID string, id int64) (*model.Project, error) {
	args := m.Called(ctx, ownerID, id)
	return args.Get(0).(*model.Project), args.Error(1)
}

func (m *MockProjectService) List(ctx context.Context, ownerID string, page, pageSize int) (*dto.PageWrapper[model.Project], error) {
	args := m.Called(ctx, ownerID, page, pageSize)
	return args.Get(0).(*dto.PageWrapper[model.Project]), args.Error(1)
}

func (m *MockProjectService) Delete(ctx context.Context, ownerID string, id int64) error {
	args := m.Called(ctx, ownerID, id)
	return args.Error(0)
}

func (m *MockProjectService) CanManage(ctx context.Context, ownerID string, id int64) (bool, error) {
	args := m.Called(ctx, ownerID, id)
	return args.Bool(0), args.Error(1)
}
