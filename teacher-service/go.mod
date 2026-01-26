module github.com/noxturnedev/lms-monorepo/teacher-service

go 1.25.5

replace github.com/noxturnedev/lms-monorepo/proto => ../proto

require (
	github.com/lib/pq v1.10.9
	github.com/noxturnedev/lms-monorepo/proto v0.0.0-00010101000000-000000000000
	github.com/rabbitmq/amqp091-go v1.10.0
	google.golang.org/grpc v1.78.0
)

require (
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
