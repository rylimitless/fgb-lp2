FROM golang:1.24-alpine

RUN apk add --no-cache gcc musl-dev git poppler-utils
RUN go install github.com/air-verse/air@v1.61.7

ENV GOTOOLCHAIN=auto

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

EXPOSE 5555

CMD ["air", "-c", ".air.toml"]
