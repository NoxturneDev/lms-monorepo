package web

import (
	schoolpb "github.com/noxturnedev/lms-monorepo/proto/school"
	statspb "github.com/noxturnedev/lms-monorepo/proto/stats"
	studentpb "github.com/noxturnedev/lms-monorepo/proto/student"
	teacherpb "github.com/noxturnedev/lms-monorepo/proto/teacher"
)

type Gateway struct {
	StudentClient studentpb.StudentServiceClient
	TeacherClient teacherpb.TeacherServiceClient
	SchoolClient  schoolpb.SchoolServiceClient
	StatsClient   statspb.StatsServiceClient
}

func NewGateway(s studentpb.StudentServiceClient, t teacherpb.TeacherServiceClient, sc schoolpb.SchoolServiceClient, st statspb.StatsServiceClient) *Gateway {
	return &Gateway{
		StudentClient: s,
		TeacherClient: t,
		SchoolClient:  sc,
		StatsClient:   st,
	}
}
