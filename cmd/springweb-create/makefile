.PHONY:
all: main.wasm wasm_exec.js
main.wasm: main.go
	@GOOS=js GOARCH=wasm go build -o $@ $^
wasm_exec.js:
	@ln -s "`go env GOROOT`/misc/wasm/wasm_exec.js" $@
.PHONY:
deploy:
	cp main.wasm wasm_exec.js ../../assets/[01].png $(WEB)/.
	@echo "Please have 'application/wasm wasm' in mime.types"
.PHONY:
clean:
	@rm -f wasm_exec.js main.wasm
