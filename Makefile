.PHONY: test vet build contracts web scripts verify release

test:
	go test ./...

vet:
	go vet ./...

build:
	go build ./...

contracts:
	forge test --root contracts

web:
	npm --prefix web test -- --run
	npm --prefix web run lint
	npm --prefix web run build

scripts:
	bash -n scripts/build-release.sh scripts/e2e-testnet.sh

verify: test vet build contracts web scripts

release:
	./scripts/build-release.sh
