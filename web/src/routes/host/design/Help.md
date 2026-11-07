# Challenge Introduction 🪄
---

Initialize a go project:
```bash
go mod init github.com/megakuul/cloudsky
go get github.com/cloudjam-foundation/cloudjam
```

Write your challenge in `main.go`:
```go
//go:build wasip1

package main

func main() {

}
```

Compile the final challenge plugin:
```
GOOS=wasip1 GOARCH=wasm go build -o cloudsky.wasm .
```
