ARG GO_VERSION=1.20.5
FROM golang:${GO_VERSION}-bullseye as build

WORKDIR /go/src/app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd cmd
RUN CGO_ENABLED=0 go build ./cmd/scheduler

FROM gcr.io/distroless/static-debian11:nonroot
COPY --from=build --chown=nonroot:. /go/src/app/scheduler /
CMD ["/scheduler"]
