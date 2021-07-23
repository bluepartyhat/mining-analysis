module main

go 1.16

replace github.com/bitclout/core => ../core

replace github.com/golang/glog => ../core/third_party/github.com/golang/glog

replace github.com/laser/go-merkle-tree => ../core/third_party/github.com/laser/go-merkle-tree

replace github.com/sasha-s/go-deadlock => ../core/third_party/github.com/sasha-s/go-deadlock

replace github.com/bitclout/backend/scripts/tools/toolslib => ../backend/scripts/tools/toolslib

require (
	github.com/bitclout/backend v1.0.7
	github.com/pkg/errors v0.9.1
)