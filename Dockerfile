FROM golang

RUN go get github.com/jcharra/go-entrisserver

EXPOSE 8888

CMD ["bin/go-entrisserver"]