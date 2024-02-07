ARG GO_VERSION=1.21.6
FROM --platform=linux/amd64 golang:${GO_VERSION} AS build
WORKDIR /src

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

ARG TARGETARCH

RUN CGO_ENABLED=0 GOARCH=amd64 go build -o /bin/server .

FROM alpine:1.22.0 AS final

RUN --mount=type=cache,target=/var/cache/apk \
    apk --update add \
        ca-certificates \
        tzdata \
        && \
        update-ca-certificates

COPY --from=build /bin/server /bin/
COPY strings.yaml /strings.yaml
COPY assets /assets
COPY .env /.env

ENTRYPOINT [ "/bin/server" ]
