#!/bin/bash

protoc -I/usr/local/include -I. \
    -I./gobot-server/vendor/ \
    -I./third_party/googleapis \
    --go_out=plugins=grpc:./gobot-server/pb \
    --grpc-gateway_out=logtostderr=true:./gobot-server/pb \
    ./server.proto
