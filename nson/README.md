# NSON

A crazy way to encode Nostr events into valid JSON that is also much faster to decode as long as they are actually
encoded using the strict NSON encoding and the decoder is prepared to read it using a NSON decoder.

See https://github.com/nostr-protocol/nips/pull/515.

Some benchmarks:

```
goos: linux
goarch: amd64
pkg: github.com/nbd-wtf/go-nostr/nson
cpu: 13th Gen Intel(R) Core(TM) i7-13620H
BenchmarkNSONEncoding/easyjson.Marshal-16                  18795             61397 ns/op
BenchmarkNSONEncoding/nson.Marshal-16                       5985            205112 ns/op
BenchmarkNSONDecoding/easyjson.Unmarshal-16                14928             83890 ns/op
BenchmarkNSONDecoding/nson.Unmarshal-16                    24982             50527 ns/op
BenchmarkNSONDecoding/easyjson.Unmarshal+sig-16              196           5898287 ns/op
BenchmarkNSONDecoding/nson.Unmarshal+sig-16                  205           5802747 ns/op
PASS
ok      github.com/nbd-wtf/go-nostr/nson        10.227s
```

It takes a little while more to encode (although it's probably possible to optimize that), but decodes at 10x the
speed of normal JSON.

The performance gain is real, but negligible once you add hash validation and signature verification, so it should
be used wisely, mostly for situations in which the reader wouldn't care about the signature, e.g. reading from a
local database.

## How it works

It's explained better in the NIP proposal linked above, but the idea is that we encode field offset and sizes into
a special JSON attribute called `"nson"`, and then the reader can just pull the strings directly from the JSON blob
without having to parse the full JSON syntax. Also for fields of static size we don't even need that. This is only
possible because Nostr events have a static and strict format.

## Update: comparison with `easyjson`

Another comparison, using the `easyjson` library that is already built in `go-nostr`, shows that the performance gains
are only of 2x (the standard library JSON encoding is just too slow).

```
goos: linux
goarch: amd64
pkg: github.com/nbd-wtf/go-nostr/nson
cpu: AMD Ryzen 3 3200G with Radeon Vega Graphics
BenchmarkNSONEncoding/easyjson.Marshal-4                   21511             54849 ns/op
BenchmarkNSONEncoding/nson.Marshal-4                        4810            297624 ns/op
BenchmarkNSONDecoding/easyjson.Unmarshal-4                 25196             46652 ns/op
BenchmarkNSONDecoding/nson.Unmarshal-4                     61117             22933 ns/op
BenchmarkNSONDecoding/easyjson.Unmarshal+sig-4               303           4110988 ns/op
BenchmarkNSONDecoding/nson.Unmarshal+sig-4                   296           3881435 ns/op
PASS
ok      github.com/nbd-wtf/go-nostr/nson
```
