.PHONY:
all: main.wasm wasm_exec.js
main.wasm: main.go
	@GOOS=js GOARCH=wasm go build -o $@ $^
wasm_exec.js:
	@ln -s "`go env GOROOT`/misc/wasm/wasm_exec.js" $@
.PHONY:
clean:
	@rm -f wasm_exec.js main.wasm
