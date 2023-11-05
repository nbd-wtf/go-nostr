build-all:
    #!/usr/bin/env fish
    for dir in (find . -maxdepth 1 -type d -name "nip*")
        go build "./$dir"
    end
    go build ./
    go build ./nson
    go build ./binary

test-all:
    #!/usr/bin/env fish
    for dir in (find . -maxdepth 1 -type d -name "nip*")
        go test "./$dir"
    end
    go test ./
    go test ./nson
    go test ./binary
