# NSON

A crazy way to encode Nostr events into valid JSON that is also much faster to decode as long as they are actually
encoded using the strict NSON encoding and the decoder is prepared to read it using a NSON decoder.

See https://github.com/nostr-protocol/nips/pull/515.

Some benchmarks:

```
goos: linux
goarch: amd64
pkg: github.com/nbd-wtf/go-nostr/nson
cpu: AMD Ryzen 3 3200G with Radeon Vega Graphics
BenchmarkNSONEncoding/json.Marshal-4                6214            230680 ns/op
BenchmarkNSONEncoding/nson.Marshal-4                4520            319058 ns/op
BenchmarkNSONDecoding/json.Unmarshal-4              3741            280641 ns/op
BenchmarkNSONDecoding/nson.Unmarshal-4             46519             23762 ns/op
BenchmarkNSONDecoding/json.Unmarshal_+_sig_verification-4                    352           3218583 ns/op
BenchmarkNSONDecoding/nson.Unmarshal_+_sig_verification-4                    451           2739238 ns/op
PASS
ok      github.com/nbd-wtf/go-nostr/nson        8.291s
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
