FROM golang:1.26.11-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /server ./cmd/server && CGO_ENABLED=0 go build -trimpath -o /seed ./cmd/seed

FROM build AS test
CMD ["sh", "-c", "go test -count=1 -v ./... && go vet ./..."]

FROM alpine:3.22
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=build /server /seed /app/
COPY backend/testdata /app/testdata
USER app
EXPOSE 8080
ENTRYPOINT ["/app/server"]
