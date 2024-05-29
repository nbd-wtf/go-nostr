This is faster than the pure Go version:

```
goos: linux
goarch: amd64
pkg: github.com/nbd-wtf/go-nostr/libsecp256k1
cpu: AMD Ryzen 3 3200G with Radeon Vega Graphics
BenchmarkSignatureVerification/btcec-4         	     145	   7873130 ns/op	  127069 B/op	     579 allocs/op
BenchmarkSignatureVerification/libsecp256k1-4  	     502	   2314573 ns/op	  112241 B/op	     392 allocs/op
```
