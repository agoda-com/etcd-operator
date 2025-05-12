# syntax=docker/dockerfile:1

ARG BASE_IMAGE=gcr.io/distroless/base-debian12:nonroot

FROM golang:1.24 AS build

WORKDIR /app
COPY . .

ARG SKAFFOLD_GO_GCFLAGS
ARG GOFLAGS
ARG GOMODCACHE=/go/pkg/mod
ARG GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target=${GOMODCACHE} \
    --mount=type=cache,target=${GOCACHE} \
    mkdir -p /etc/etcd/bin && \
    go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o /etc/etcd/bin/ ./cmd/... && \
    go test -o /etc/etcd/bin/etcd-e2e-test -c ./e2e

FROM $BASE_IMAGE
COPY --from=build /etc/etcd/bin /etc/etcd/bin

ENV GOTRACEBACK=all
ENV PATH=${PATH}:/etc/etcd/bin
ENTRYPOINT [ "/etc/etcd/bin/etcd-operator" ]
