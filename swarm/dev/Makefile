.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/teamnsrg/ethereum-p2p/cmd/swarm
	go build -o bin/geth     github.com/teamnsrg/ethereum-p2p/cmd/geth
	go build -o bin/bootnode github.com/teamnsrg/ethereum-p2p/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/teamnsrg/ethereum-p2p/swarm/...
