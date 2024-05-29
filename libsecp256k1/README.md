This wraps [libsecp256k1](https://github.com/bitcoin-core/secp256k1) with `cgo`.

It doesn't embed the library or anything smart like that because I don't know how to do it, so you must have it installed in your system.

It is faster than the pure Go version:

```
goos: linux
goarch: amd64
pkg: github.com/nbd-wtf/go-nostr/libsecp256k1
cpu: AMD Ryzen 3 3200G with Radeon Vega Graphics
BenchmarkSignatureVerification/btcec-4         	     145	   7873130 ns/op	  127069 B/op	     579 allocs/op
BenchmarkSignatureVerification/libsecp256k1-4  	     502	   2314573 ns/op	  112241 B/op	     392 allocs/op
```

To use it manually, just import. To use it inside the automatic verification that happens for subscriptions, set it up with a `SimplePool`:

```go
pool := nostr.NewSimplePool()
pool.SignatureChecker = func (evt nostr.Event) bool {
	ok, _ := libsecp256k1.CheckSignature(evt)
	return ok
}
```

Or directly to the `Relay`:

```go
relay := nostr.RelayConnect(context.Background(), "wss://relay.nostr.com", nostr.WithSignatureChecker(func (evt nostr.Event) bool {
	ok, _ := libsecp256k1.CheckSignature(evt)
	return ok
}))
```
