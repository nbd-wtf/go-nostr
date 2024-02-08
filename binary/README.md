# The simplest binary encoding for Nostr events

Some benchmarks:

```
goos: linux
goarch: amd64
pkg: github.com/nbd-wtf/go-nostr/binary
cpu: AMD Ryzen 3 3200G with Radeon Vega Graphics
BenchmarkBinaryEncoding/easyjson.Marshal-4         	    8283	    153034 ns/op	   95167 B/op	     123 allocs/op
BenchmarkBinaryEncoding/gob.Encode-4               	    3601	    299684 ns/op	  407859 B/op	    1491 allocs/op
BenchmarkBinaryEncoding/binary.Marshal-4           	    4004	    346269 ns/op	 1476069 B/op	     814 allocs/op
BenchmarkBinaryEncoding/binary.MarshalBinary-4     	    3368	    354479 ns/op	 1471205 B/op	     757 allocs/op
BenchmarkBinaryDecoding/easyjson.Unmarshal-4       	    4684	    253556 ns/op	  257561 B/op	    1584 allocs/op
BenchmarkBinaryDecoding/gob.Decode-4               	    1311	    922829 ns/op	  427914 B/op	    7883 allocs/op
BenchmarkBinaryDecoding/binary.Unmarshal-4         	   13438	     89201 ns/op	  114576 B/op	    1592 allocs/op
BenchmarkBinaryDecoding/binary.UnmarshalBinary-4   	   14200	     84410 ns/op	  104848 B/op	    1478 allocs/op
BenchmarkBinaryDecoding/easyjson.Unmarshal+sig-4   	     259	   4720044 ns/op	  588309 B/op	    1920 allocs/op
BenchmarkBinaryDecoding/binary.Unmarshal+sig-4     	     271	   4514978 ns/op	  445286 B/op	    1928 allocs/op
PASS
ok  	github.com/nbd-wtf/go-nostr/binary	15.109s
```

This is 2~5x faster than [NSON](../nson) decoding, which means 8x faster than default easyjson decoding,
but, just like NSON, the performance gains from this encoding is negligible when you add the cost of
signature verification. Which means this encoding must only be used in internal processes.
