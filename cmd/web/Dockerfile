##################################
# STEP 1 build executable binary #
##################################
FROM golang:alpine AS builder

# Install git.
# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git
COPY . $GOPATH/src/github.com/scotow/goutube

# Move to command directory.
WORKDIR $GOPATH/src/github.com/scotow/goutube/cmd/web

# Fetch dependencies.
# Using go get.
RUN go get -d -v

# Build the binary.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/goutube

##############################
# STEP 2 build a small image #
##############################
FROM alpine

RUN apk update && apk add --no-cache youtube-dl

# Copy our static executable.
COPY --from=builder /go/bin/goutube /goutube

# Run the hello binary.
ENTRYPOINT ["/goutube"]