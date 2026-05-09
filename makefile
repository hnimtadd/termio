.PHONY: run
run:
	@go run examples/terminal/main.go
.PHONY: vet
vet:
	@go vet ./...
