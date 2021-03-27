FROM golang:1.15 AS builder

WORKDIR /app

COPY go.mod go.sum ./

# Extremely imperfect means of installing packages, but helps with Docker
#   build times
RUN go get $(grep -zo 'require (\(.*\))' go.mod | sed '1d;$d;' | tr ' ' '@') 

COPY . .

# https://stackoverflow.com/a/62123648
RUN CGO_ENABLED=0 make


FROM alpine AS runner

COPY --from=builder /app/build/rmqhttp /usr/bin/rmqhttp

ENTRYPOINT ["/usr/bin/rmqhttp"]
CMD ["server"]
