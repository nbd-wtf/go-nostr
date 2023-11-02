# The simplest binary encoding for Nostr events

Some benchmarks:

goos: linux
goarch: amd64
pkg: github.com/nbd-wtf/go-nostr/binary
cpu: AMD Ryzen 3 3200G with Radeon Vega Graphics
BenchmarkBinaryEncoding/easyjson.Marshal-4                 24488             53274 ns/op           35191 B/op        102 allocs/op
BenchmarkBinaryEncoding/binary.Marshal-4                    5066            218284 ns/op         1282116 B/op         88 allocs/op
BenchmarkBinaryEncoding/binary.MarshalBinary-4              5743            191603 ns/op         1277763 B/op         37 allocs/op
BenchmarkBinaryDecoding/easyjson.Unmarshal-4               32701             38647 ns/op           45832 B/op        124 allocs/op
BenchmarkBinaryDecoding/binary.Unmarshal-4                 85705             14249 ns/op           25488 B/op        141 allocs/op
BenchmarkBinaryDecoding/binary.UnmarshalBinary-4          213438              5451 ns/op           16784 B/op         39 allocs/op
BenchmarkBinaryDecoding/easyjson.Unmarshal+sig-4             307           3971993 ns/op          131639 B/op        404 allocs/op
BenchmarkBinaryDecoding/binary.Unmarshal+sig-4               310           3924042 ns/op          111277 B/op        421 allocs/op
PASS
ok      github.com/nbd-wtf/go-nostr/binary      11.444s
