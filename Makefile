.PHONY: test vet build contracts verify

test:
	go test ./...

vet:
	go vet ./...

build:
	go build ./...

contracts:
	forge test --root contracts

verify: test vet build contracts
