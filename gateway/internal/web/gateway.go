package web

import (
	studentpb "github.com/noxturnedev/lms-monorepo/proto/student"
	teacherpb "github.com/noxturnedev/lms-monorepo/proto/teacher"
)

type Gateway struct {
	StudentClient studentpb.StudentServiceClient
	TeacherClient teacherpb.TeacherServiceClient
}

func NewGateway(s studentpb.StudentServiceClient, t teacherpb.TeacherServiceClient) *Gateway {
	return &Gateway{
		StudentClient: s,
		TeacherClient: t,
	}
}
