#!/bin/bash
go build -o singgen ./cmd/singgen
./singgen -config test-config.yaml -out test-config.json