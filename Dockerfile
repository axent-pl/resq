FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/resq .

FROM alpine:3.22

WORKDIR /app

COPY --from=build /out/resq /app/resq
COPY templates /app/templates
COPY data /app/data

VOLUME ["/app/data"]
EXPOSE 1234

ENV RP_DISPLAY_NAME=RESQ
ENV RP_ID=RESQ
ENV RP_ORIGINS=http://localhost:1234

ENTRYPOINT ["/app/resq"]
