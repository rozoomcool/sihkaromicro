package repository

import (
	"context"

	"github.com/rozoomcool/sihkaromicro/sources/internal/apperr"
	"github.com/rozoomcool/sihkaromicro/sources/internal/model"
	"gorm.io/gorm"
)

type SourceRepository interface {
	Save(ctx context.Context, source *model.Source) error
	FindByProjectIDAndOwnerID(ctx context.Context, id, projectID int64, ownerID string) (*model.Source, error)
	FindAllByProjectIDAndOwnerID(ctx context.Context, projectID int64, ownerID string) ([]model.Source, error)
	CountByProjectIDAndOwnerID(ctx context.Context, projectID int64, ownerID string) (int64, error)
	// UpdateStatusBySourceID updates status and job_id for the given source ID.
	UpdateStatusBySourceID(ctx context.Context, id int64, status model.SourceStatus, jobID string) error
	UpdateMinioPath(ctx context.Context, id int64, minioPath string) error
	DeleteByProjectIDAndOwnerID(ctx context.Context, id, projectID int64, ownerID string) error
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

func (r *sourceRepository) FindByProjectIDAndOwnerID(ctx context.Context, id, projectID int64, ownerID string) (*model.Source, error) {
	source := new(model.Source)
	err := r.db.WithContext(ctx).
		Where("id = ? AND project_id = ? AND owner_id = ?", id, projectID, ownerID).
		First(source).Error
	if err != nil {
		return nil, apperr.FromGORM(err, "source FindByProjectIDAndOwnerID")
	}
	return source, nil
}

func (r *sourceRepository) UpdateStatusBySourceID(ctx context.Context, id int64, status model.SourceStatus, jobID string) error {
	return apperr.FromGORM(r.db.WithContext(ctx).Model(&model.Source{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status": status,
			"job_id": jobID,
		}).Error, "source UpdateStatusBySourceID")
}

func (r *sourceRepository) DeleteByProjectIDAndOwnerID(ctx context.Context, id, projectID int64, ownerID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND project_id = ? AND owner_id = ?", id, projectID, ownerID).
		Delete(&model.Source{})
	if result.Error != nil {
		return apperr.FromGORM(result.Error, "source DeleteByProjectIDAndOwnerID")
	}
	if result.RowsAffected == 0 {
		return apperr.FromGORM(gorm.ErrRecordNotFound, "source DeleteByProjectIDAndOwnerID")
	}
	return nil
}

func (r *sourceRepository) FindAllByProjectIDAndOwnerID(ctx context.Context, projectID int64, ownerID string) ([]model.Source, error) {
	var sources []model.Source
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND owner_id = ?", projectID, ownerID).
		Find(&sources).Error
	if err != nil {
		return nil, apperr.FromGORM(err, "source FindAllByProjectIDAndOwnerID")
	}
	return sources, nil
}

func (r *sourceRepository) CountByProjectIDAndOwnerID(ctx context.Context, projectID int64, ownerID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Source{}).
		Where("project_id = ? AND owner_id = ?", projectID, ownerID).
		Count(&count).Error
	if err != nil {
		return 0, apperr.FromGORM(err, "source CountByProjectIDAndOwnerID")
	}
	return count, nil
}

func (r *sourceRepository) UpdateMinioPath(ctx context.Context, id int64, minioPath string) error {
	return apperr.FromGORM(r.db.WithContext(ctx).
		Model(&model.Source{}).
		Where("id = ?", id).
		Update("minio_path", minioPath).Error, "source UpdateMinioPath")
}
