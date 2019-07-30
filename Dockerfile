############################
# STEP 1 build executable binary
############################
FROM golang:alpine as builder
# Install git + SSL ca certificates.
# Git is required for fetching the dependencies.
# Ca-certificates is required to call HTTPS endpoints.
RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates
# Create appuser
RUN adduser -D -g '' app
WORKDIR $GOPATH/src/github.com/nickethier/cf-ddns
COPY . .
# Fetch dependencies.
# Using go get.
RUN go get -d -v
# Using go mod.
# RUN go mod download
# RUN go mod verify
# Build the binary
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/cfddns
############################
# STEP 2 build a small image
############################
FROM alpine
RUN apk add -U --no-cache ca-certificates tzdata && update-ca-certificates
# Import from builder.
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
# Copy our static executable
COPY --from=builder /go/bin/cfddns /bin/cfddns
# Use an unprivileged user.
USER app
# Run the hello binary.
ENTRYPOINT ["/bin/cfddns"]
