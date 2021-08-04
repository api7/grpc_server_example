.PHONY: package
package:
	CGO_ENABLED=0 go build
	tar -czvf grpc_server_example-amd64.tar.gz grpc_server_example
