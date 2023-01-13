<a href="https://nbd.wtf"><img align="right" height="196" src="https://user-images.githubusercontent.com/1653275/194609043-0add674b-dd40-41ed-986c-ab4a2e053092.png" /></a>

go-nostr
========

A set of useful things for [Nostr Protocol](https://github.com/nostr-protocol/nostr) implementations.

<a href="https://godoc.org/github.com/nbd-wtf/go-nostr"><img src="https://img.shields.io/badge/api-reference-blue.svg?style=flat-square" alt="GoDoc"></a>

### Generating a key

``` go
sk, _ := nostr.GenerateKey()

fmt.Println("sk:", sk)
fmt.Println("pk:", nostr.GetPublicKey(sk))
```
