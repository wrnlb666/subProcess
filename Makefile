CC ?= musl-gcc


main: 
	CC=$(CC) go build --ldflags '-linkmode external -extldflags "-static"'
