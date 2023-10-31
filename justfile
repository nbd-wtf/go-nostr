build-all:
    #!/usr/bin/env fish
    for dir in (find . -maxdepth 1 -type d -name "nip*")
        echo "building $dir"
        go build "./$dir"
    end

test-all:
    #!/usr/bin/env fish
    for dir in (find . -maxdepth 1 -type d -name "nip*")
        echo "testing $dir"
        go test "./$dir"
    end
