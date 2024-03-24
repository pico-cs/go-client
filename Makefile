# builds and tests project via go tools
all:
	@echo "update dependencies"
	go get -u ./...
	go mod tidy
	@echo "build and test"
	go build -v ./...
	go vet ./...
	golint -set_exit_status=true ./...
	staticcheck -checks all -fail none ./...
	golangci-lint run ./...
	go test ./...
#see fsfe reuse tool (https://git.fsfe.org/reuse/tool)
	@echo "reuse (license) check"
	pipx run reuse lint

#go generate
generate:
	@echo "generate"
	go generate ./...

#install additional tools
tools:
#install stringer
	@echo "install latest stringer version"
	go install golang.org/x/tools/cmd/stringer@latest
#install linter
	@echo "install latest go linter version"
	go install golang.org/x/lint/golint@latest
#install staticcheck
	@echo "install latest staticcheck version"
	go install honnef.co/go/tools/cmd/staticcheck@latest
#install golangci-lint
	@echo "install latest golangci-lint version"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
