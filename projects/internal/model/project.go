package model

import (
	pb "github.com/rozoomcool/sihkaromicro/proto/projects"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Project struct {
	Base
	OwnerID string `gorm:"column:owner_id;type:char(36);not null;index"`
	Title   string `gorm:"column:title;not null;size:255" json:"title"`
}

func (Project) TableName() string {
	return "projects"
}

func (p *Project) ToProto() *pb.Project {
	return &pb.Project{
		Id:        p.ID,
		OwnerId:   p.OwnerID,
		Title:     p.Title,
		CreatedAt: timestamppb.New(p.CreatedAt),
		UpdatedAt: timestamppb.New(p.UpdatedAt),
	}
}
