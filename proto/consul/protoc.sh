#!/bin/bash

#protoc -I ../consul --go_out=plugins=grpc:consul ../consul/consul.proto
protoc -I . --go_out=plugins=grpc:.  consul.proto
