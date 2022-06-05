FROM golang:1.18.3-bullseye AS golang

WORKDIR /

COPY go.mod go.sum *.csv ./
RUN go mod download
COPY . .

RUN go build -o /go/bin/app main.go


# Now copy it into our base image.
FROM gcr.io/distroless/base-debian11
COPY --from=golang /go/bin/app /
COPY --from=golang /*.csv /
ENTRYPOINT [ "/app" ]CMD []