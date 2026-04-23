package repository

import (
	"context"

	"github.com/rozoomcool/sihkaromicro/sources/internal/model"
	"gorm.io/gorm"
)

type SourceRepository interface {
	Save(ctx context.Context, source *model.Source) error
	Find(ctx context.Context, id, projectID int64, ownerID string) (*model.Source, error)
	FindAll(ctx context.Context, projectID int64, ownerID string) ([]model.Source, error)
	Count(ctx context.Context, projectID int64, ownerID string) (int64, error)
	UpdateStatus(ctx context.Context, id int64, status model.SourceStatus, jobID string) error
	Delete(ctx context.Context, id, projectID int64, ownerID string) error
}

type sourceRepository struct {
	db *gorm.DB
}

func NewSourceRepository(db *gorm.DB) SourceRepository {
	return &sourceRepository{db: db}
}

func (r *sourceRepository) Save(ctx context.Context, source *model.Source) error {
	return r.db.WithContext(ctx).Create(source).Error
}

func (r *sourceRepository) Find(ctx context.Context, id, projectID int64, ownerID string) (*model.Source, error) {
	var source model.Source
	err := r.db.WithContext(ctx).
		Where("id = ? AND project_id = ? AND owner_id = ?", id, projectID, ownerID).
		First(&source).Error
	if err != nil {
		return nil, err
	}
	return &source, nil
}

func (r *sourceRepository) UpdateStatus(ctx context.Context, id int64, status model.SourceStatus, jobID string) error {
	return r.db.WithContext(ctx).Model(&model.Source{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status": status,
			"job_id": jobID,
		}).Error
}

func (r *sourceRepository) Delete(ctx context.Context, id, projectID int64, ownerID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND project_id = ? AND owner_id = ?", id, projectID, ownerID).
		Delete(&model.Source{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *sourceRepository) FindAll(ctx context.Context, projectID int64, ownerID string) ([]model.Source, error) {
	var sources []model.Source
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND owner_id = ?", projectID, ownerID).
		Find(&sources).Error
	if err != nil {
		return nil, err
	}
	return sources, nil
}

func (r *sourceRepository) Count(ctx context.Context, projectID int64, ownerID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Source{}).
		Where("project_id = ? AND owner_id = ?", projectID, ownerID).
		Count(&count).Error
	return count, err
}
