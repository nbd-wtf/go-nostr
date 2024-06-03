# The simplest binary encoding for Nostr events

Some benchmarks:

```
goos: linux
goarch: amd64
pkg: github.com/nbd-wtf/go-nostr/binary
cpu: AMD Ryzen 3 3200G with Radeon Vega Graphics
BenchmarkBinaryEncoding/easyjson.Marshal-4         	   12756	    109437 ns/op	   66058 B/op	     227 allocs/op
BenchmarkBinaryEncoding/gob.Encode-4               	    3807	    367426 ns/op	  171456 B/op	    1501 allocs/op
BenchmarkBinaryEncoding/binary.Marshal-4           	    2568	    486766 ns/op	 2736133 B/op	      37 allocs/op
BenchmarkBinaryEncoding/binary.MarshalBinary-4     	    2150	    525876 ns/op	 2736135 B/op	      37 allocs/op
BenchmarkBinaryDecoding/easyjson.Unmarshal-4       	   13719	     92516 ns/op	   82680 B/op	     360 allocs/op
BenchmarkBinaryDecoding/gob.Decode-4               	     938	   1469278 ns/op	  386459 B/op	    8549 allocs/op
BenchmarkBinaryDecoding/binary.Unmarshal-4         	   49454	     29724 ns/op	   21776 B/op	     282 allocs/op
BenchmarkBinaryDecoding/binary.UnmarshalBinary-4   	  230827	      6876 ns/op	    2832 B/op	      60 allocs/op
BenchmarkBinaryDecoding/easyjson.Unmarshal+sig-4   	     177	   7038434 ns/op	  209834 B/op	     939 allocs/op
BenchmarkBinaryDecoding/binary.Unmarshal+sig-4     	     180	   6727125 ns/op	  148841 B/op	     861 allocs/op
PASS
ok  	github.com/nbd-wtf/go-nostr/binary	16.937s
```

This is 2~5x faster than [NSON](../nson) decoding, which means 8x faster than default easyjson decoding,
but, just like NSON, the performance gains from this encoding is negligible when you add the cost of
signature verification. Which means this encoding must only be used in internal processes.
