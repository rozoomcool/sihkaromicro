package model

import (
	"time"

	pb "github.com/rozoomcool/sihkaromicro/proto/sources"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SourceStatus string
type SourceType string

const (
	StatusUploading  SourceStatus = "uploading"
	StatusUploaded   SourceStatus = "uploaded"
	StatusPending    SourceStatus = "pending"
	StatusProcessing SourceStatus = "processing"
	StatusReady      SourceStatus = "ready"
	StatusFailed     SourceStatus = "failed"
)

const (
	TypePDF      SourceType = "pdf"
	TypeTXT      SourceType = "txt"
	TypeDOCX     SourceType = "docx"
	TypeMarkdown SourceType = "markdown"
	TypeURL      SourceType = "url"
)

type Source struct {
	ID        int64        `gorm:"column:id;primaryKey;autoIncrement"`
	ProjectID int64        `gorm:"column:project_id;not null;index"`
	OwnerID   string       `gorm:"column:owner_id;not null;index"`
	Name      string       `gorm:"column:name;not null"`
	Type      SourceType   `gorm:"column:type;not null"`
	Status    SourceStatus `gorm:"column:status;not null;default:uploading"`
	Size      int64        `gorm:"column:size;not null"`
	MinioPath string       `gorm:"column:minio_path"`
	JobID     string       `gorm:"column:job_id"`
	Error     string       `gorm:"column:error"`
	SourceURL string       `gorm:"column:source_url"`
	CreatedAt time.Time    `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time    `gorm:"column:updated_at;autoUpdateTime"`
}

func (s *Source) ToProto() *pb.Source {
	return &pb.Source{
		Id:        s.ID,
		ProjectId: s.ProjectID,
		OwnerId:   s.OwnerID,
		Name:      s.Name,
		Type:      sourceTypeToProto(s.Type),
		Status:    sourceStatusToProto(s.Status),
		Size:      s.Size,
		JobId:     s.JobID,
		Error:     s.Error,
		SourceUrl: s.SourceURL,
		CreatedAt: timestamppb.New(s.CreatedAt),
		UpdatedAt: timestamppb.New(s.UpdatedAt),
	}
}

func sourceTypeToProto(t SourceType) pb.SourceType {
	switch t {
	case TypePDF:
		return pb.SourceType_SOURCE_TYPE_PDF
	case TypeTXT:
		return pb.SourceType_SOURCE_TYPE_TXT
	case TypeDOCX:
		return pb.SourceType_SOURCE_TYPE_DOCX
	case TypeMarkdown:
		return pb.SourceType_SOURCE_TYPE_MARKDOWN
	case TypeURL:
		return pb.SourceType_SOURCE_TYPE_URL
	default:
		return pb.SourceType_SOURCE_TYPE_UNSPECIFIED
	}
}

func sourceStatusToProto(s SourceStatus) pb.SourceStatus {
	switch s {
	case StatusUploading:
		return pb.SourceStatus_SOURCE_STATUS_UPLOADING
	case StatusUploaded:
		return pb.SourceStatus_SOURCE_STATUS_UPLOADED
	case StatusPending:
		return pb.SourceStatus_SOURCE_STATUS_PENDING
	case StatusProcessing:
		return pb.SourceStatus_SOURCE_STATUS_PROCESSING
	case StatusReady:
		return pb.SourceStatus_SOURCE_STATUS_READY
	case StatusFailed:
		return pb.SourceStatus_SOURCE_STATUS_FAILED
	default:
		return pb.SourceStatus_SOURCE_STATUS_UNSPECIFIED
	}
}
