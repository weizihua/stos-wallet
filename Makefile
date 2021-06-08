.PHONY: manager

wallet:ffi
	go build
docker:
	git pull && docker build --tag stos-wallet -f ./Dockerfile .

ffi:
	-git rm --cached extern/filecoin-ffi
	-git submodule add https://github.com/filecoin-project/filecoin-ffi.git extern/filecoin-ffi
	-make -C extern/filecoin-ffi
	-go mod edit -replace=github.com/filecoin-project/filecoin-ffi=./extern/filecoin-ffi

