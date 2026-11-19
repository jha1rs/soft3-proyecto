# ---- Compilacion ----
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /api .

# ---- Imagen final: solo el binario (~8 MB) ----
FROM scratch
COPY --from=build /api /api
USER 65534:65534
EXPOSE 8080
ENTRYPOINT ["/api"]
