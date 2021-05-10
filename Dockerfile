FROM golang:1.15 as build

# Add the local files to be sure we are building the local source code instead
# of downloading from GitHub. All other package dependencies will be downloaded
# from HEAD.
ADD . /go/src/github.com/m-lab/epoxy
ENV CGO_ENABLED 0
WORKDIR /go/src/github.com/m-lab/epoxy
RUN go get -t -v ./...
RUN go test -v ./...
RUN go get \
      -v \
      -ldflags "-X github.com/m-lab/go/prometheusx.GitShortCommit=$(git log -1 --format=%h)" \
      ./...

# Now copy the built binary into a minimal base image.
FROM alpine
COPY --from=build /go/bin/epoxy_boot_server /

# We must install the ca-certificates package so the ePoxy server can securely
# connect to the LetsEncrypt servers to register & create our certificates.
# As well, valid ca-certificates are needed for the storage proxy connections.
RUN apk add --no-cache ca-certificates && update-ca-certificates

WORKDIR /
ENTRYPOINT ["/epoxy_boot_server"]
