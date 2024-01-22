FROM --platform=$BUILDPLATFORM golang AS build

ARG GIT_DESC=undefined

WORKDIR /go/src/github.com/Snawoot/rgap
COPY . .
ARG TARGETOS TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -a -tags netgo -ldflags '-s -w -extldflags "-static" -X main.version='"$GIT_DESC" ./cmd/rgap

FROM scratch
COPY --from=build /go/src/github.com/Snawoot/rgap/rgap /
USER 9999:9999
ENTRYPOINT ["/rgap"]
