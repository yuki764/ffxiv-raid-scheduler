ARG GO_VERSION
FROM golang:${GO_VERSION}-trixie AS build

WORKDIR /go/src/app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd cmd
RUN CGO_ENABLED=0 go build ./cmd/scheduler

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build --chown=nonroot:. /go/src/app/scheduler /
CMD ["/scheduler"]
