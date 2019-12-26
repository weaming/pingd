FROM golang:1.13-alpine3.10
WORKDIR /app
ENV GOPATH /go
ENV GOBIN /go/bin
ENV APP_ROOT /go/src/github.com/weaming/pingd

COPY . $APP_ROOT
RUN cd $APP_ROOT && go get && go build -ldflags "-s -w" -o /app/app ./examples/http-redis-hub

CMD /app/app
