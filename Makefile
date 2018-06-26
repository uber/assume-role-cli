.PHONY: all
all: assume-role

.PHONY: clean
clean:
	rm -f ./assume-role

.PHONY: clean-mocks
clean-mocks:
	rm mocks/*.go

assume-role: vendor
	go build -o assume-role ./cli/assume-role/main.go

.PHONY: mocks
mocks:
	go get -u github.com/golang/mock/gomock
	go get -u github.com/golang/mock/mockgen
	mockgen -package=mocks github.com/uber/assume-role AWSProvider,AWSConfigProvider | sed -e 's/assume_role/assumerole/g' > mocks/aws_mocks.go

.PHONY: test
test: vendor
	go test -short -coverprofile=.coverage.out ./...

vendor:
	dep ensure
