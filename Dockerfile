FROM golang:1.24.3-alpine3.21 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o borque

#------------------#

FROM gcr.io/distroless/static-debian12 AS release

COPY --from=build /app/borque /usr/local/bin/borque

USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/borque"]