FROM golang:1.22-alpine AS builder

WORKDIR /usr/src/app

#RUN --mount=type=cache,target=/var/cache/apk apk add protobuf-dev git
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=cache,target="/root/.cache/go-build" \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 go build -v -ldflags="-s -w" -o /build/output

FROM scratch

WORKDIR /

COPY --from=builder /build/output /server
#RUN setcap cap_net_bind_service=+ep /coredns

USER 65052:65052
WORKDIR /data

CMD ["/server"]
