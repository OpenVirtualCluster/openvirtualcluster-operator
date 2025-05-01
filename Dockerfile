# Build the manager binary
FROM golang:1.24 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager cmd/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM alpine:3.19

WORKDIR /
COPY --from=builder /workspace/manager .

# Install Helm
ENV HELM_VERSION="v3.14.4"
RUN apk add --no-cache curl tar && \
    curl -fsSL -o helm.tar.gz https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz && \
    tar -xzf helm.tar.gz && \
    mv linux-amd64/helm /usr/local/bin/ && \
    rm -rf helm.tar.gz linux-amd64 && \
    apk del curl tar

USER 65532:65532

ENTRYPOINT ["/manager"]
