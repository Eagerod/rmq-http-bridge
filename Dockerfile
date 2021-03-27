FROM golang:1.15

WORKDIR /app

COPY go.mod go.sum ./

# Extremely imperfect means of installing packages, but helps with Docker
#   build times
RUN go get $(grep -zo 'require (\(.*\))' go.mod | sed '1d;$d;' | tr ' ' '@') 

COPY . .

RUN make install

ENTRYPOINT ["/app/build/rmqhttp"]
CMD "server"
