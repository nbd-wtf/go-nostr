build-all:
    #!/usr/bin/env bash
    for dir in $(find . -maxdepth 1 -type d -name "nip*"); do
        go build "./$dir"
    done
    go build ./
    go build ./nson
    go build ./binary

test-all:
    #!/usr/bin/env bash
    for dir in $(find . -maxdepth 1 -type d -name "nip*"); do
        go test "./$dir"
    done
    go test ./
    go test ./nson
    go test ./binary
