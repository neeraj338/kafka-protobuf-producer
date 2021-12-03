#!/bin/bash

protoc \
--go_out=./generated \
--include_imports \
--descriptor_set_out=./proto/descriptor.pb \
 ./proto/*.proto