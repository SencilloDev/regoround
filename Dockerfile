FROM golang:alpine as builder
WORKDIR /app
ENV IMAGE_TAG=dev
RUN apk update && apk upgrade && apk add --no-cache ca-certificates git
RUN update-ca-certificates
ADD . /app/
ARG VERSION
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -ldflags="-s -w -X 'github.com/SencilloDev/regoround/cmd.Version=${VERSION}'" -installsuffix cgo -o regoroundctl .

FROM builder AS tester
RUN go install github.com/fzipp/gocyclo/cmd/gocyclo@latest

FROM scratch

COPY --from=builder /app/regoroundctl .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["./regoroundctl"]
