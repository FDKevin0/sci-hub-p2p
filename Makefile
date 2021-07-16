MAKEFLAGS += --no-builtin-rules
build: Windows Linux macOS

Windows: dist/sci-hub_windows_64.exe
Linux: dist/sci-hub_linux_64
macOS: dist/sci-hub_macos_64


dist/sci-hub_macos_64:
	env GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o $@

dist/sci-hub_linux_64:
	env GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o $@

dist/sci-hub_windows_64.exe:
	env GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o $@

testdata/sm_00900000-00999999.torrent:
	bash ./fetch.bash

testdata: testdata/sm_00900000-00999999.torrent

test: testdata
	go test -covermode=atomic -coverprofile=coverage.out ./...

coverage.out: test

coverage: coverage.out

clean:
	rm dist -rf
	rm -rf ./dist \
		  ./testdata/sm_00900000-00999999.torrent \
		  ./coverage.out \
		  ./out

.PHONY: Windows Linux macOS test coverage clean testdata