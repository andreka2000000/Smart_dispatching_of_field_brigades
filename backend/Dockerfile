FROM golang:1.26.11-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY data /data
RUN CGO_ENABLED=0 go build -trimpath -o /server ./cmd/server && CGO_ENABLED=0 go build -trimpath -o /seed ./cmd/seed && CGO_ENABLED=0 go build -trimpath -o /importer ./cmd/importer && CGO_ENABLED=0 go build -trimpath -o /benchmark ./cmd/benchmark && CGO_ENABLED=0 go build -trimpath -o /accounts ./cmd/accounts

FROM build AS test
CMD ["sh", "-c", "go test -count=1 -v ./... && go vet ./..."]

FROM alpine:3.22
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=build /server /seed /importer /benchmark /accounts /app/
COPY backend/testdata /app/testdata
COPY data /app/data
USER app
EXPOSE 8080
ENTRYPOINT ["/app/server"]
