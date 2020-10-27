#!/bin/bash

go test ./... -coverprofile=storage/coverage/hade.out 
go tool cover -html=storage/coverage/hade.out -o storage/coverage/index.html
open storage/coverage/index.html
