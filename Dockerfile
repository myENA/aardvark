FROM alpine:latest AS build
ENV GOPATH /opt/go
ENV SRCPATH $GOPATH/src/github.com/myENA/aardvark
COPY . $SRCPATH
RUN \
	apk add --no-cache --no-progress ca-certificates bash git go musl-dev && \
	mkdir -p $GOPATH/bin && \
	export PATH=$GOPATH/bin:$PATH && \
	cd $SRCPATH && \
	chmod +x build/build.sh && \
	build/build.sh && \
	mv aardvark /usr/local/bin/aardvark && \
	apk del --no-cache --no-progress --purge bash git go musl-dev && \
	rm -rf $GOPATH /tmp/*

FROM alpine:latest
COPY --from=build /usr/local/bin/aardvark /usr/local/bin/aardvark
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENTRYPOINT ["/usr/local/bin/aardvark"]
