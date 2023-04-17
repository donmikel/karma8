# karma8

### Build

To build all Go binaries run:

    make build

To build docker image run:

    make build_docker

### Run locally

    make up

It will start the server.

### Test
Upload any file you want

    curl -X PUT -F file=@any.file 'http://127.0.0.1:8002/file'

Download it.

    curl 'http://127.0.0.1:8002/file/any.file' > any.file

We can also look at logs:

    docker-compose -f docker/docker-compose.yml logs -f