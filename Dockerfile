ARG GO_VERSION=1.25.1

FROM registry.docker.com/library/golang:$GO_VERSION-alpine AS base

# app lives here
WORKDIR /app


# Throw-away build stage to reduce size of final image
FROM base AS build

# Install packages needed to build
RUN apk update -qq && \
    apk add --no-cache git

COPY . .

RUN go build -o upser .

ENTRYPOINT ["/app/upser"]

CMD ["app/upser]
