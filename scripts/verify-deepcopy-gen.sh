#!/bin/bash -e

go install k8s.io/code-generator/cmd/deepcopy-gen
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

deepcopy-gen --input-dirs github.com/kong/deck/konnect \
  -O zz_generated.deepcopy \
  --go-header-file scripts/header-template.go.tmpl \
  --output-base $TMP_DIR

diff -Naur $TMP_DIR/github.com/kong/deck/konnect/zz_generated.deepcopy.go \
  konnect/zz_generated.deepcopy.go
