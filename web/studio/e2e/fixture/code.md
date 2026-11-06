# Code

```go id=r title="main.go" window=mac numbers
func main() {
	send()
}
```

```go id=r title="main.go" window=mac numbers
func main() {
	if err := send(); err != nil {
		retry()
	}
}
```
