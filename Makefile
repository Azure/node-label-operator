
# Image URL to use all building/pushing image targets
IMG ?= controller:latest
EXTRA_ARGS :=

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager
.PHONY: all

# Run tests
test: generate fmt vet
	go test ./controller/... ./azure/... -coverprofile cover.out
.PHONY: test

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go
.PHONY: manager

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet
	go run ./main.go
.PHONY: run

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kustomize build config/default | kubectl apply -f -
.PHONY: deploy

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..."
.PHONY: manifests

# Run go fmt against code
fmt:
	go fmt ./...
.PHONY: fmt

# Run go vet against code
vet:
	go vet ./...
.PHONY: vet

lint:
	golangci-lint run -j 2 $(EXTRA_ARGS)
.PHONY: lint

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt
.PHONY: generate

# Build the docker image
docker-build: # test
	docker build . -t ${IMG}
	@echo "updating kustomize image patch file for manager resource"
	sed -i'' -e 's@image: .*@image: '"${IMG}"'@' ./config/default/manager_image_patch.yaml
.PHONY: docker-build

# Push the docker image
docker-push:
	docker push ${IMG}
.PHONY: docker-push

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.0-beta.4
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
.PHONY: controller-gen
