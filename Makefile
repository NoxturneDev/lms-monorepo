# Makefile

# Variables
PROTO_DIR = proto
GO_OUT_DIR = proto

# The magic command
gen:
	@echo "Generating Go code from Proto files..."
	
	# 1. Generate Student Proto
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       $(PROTO_DIR)/student.proto
	
	# 2. Generate Teacher Proto
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       $(PROTO_DIR)/teacher.proto

	@echo "Done! The Beast has been tamed."
