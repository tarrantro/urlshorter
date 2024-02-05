FROM golang:1.21.6

COPY urlshorter /urlshorter

ENV HOME /

ENTRYPOINT ["/urlshorter"]