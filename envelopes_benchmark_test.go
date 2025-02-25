package nostr

import (
	"bufio"
	"os"
	"testing"

	"github.com/minio/simdjson-go"
)

// benchmarkParseMessage benchmarks the ParseMessage function
func BenchmarkParseMessage(b *testing.B) {
	messages := getTestMessages()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, msg := range messages {
			_ = ParseMessage(msg)
		}
	}
}

// benchmarkParseMessageSIMD benchmarks the ParseMessageSIMD function
func BenchmarkParseMessageSIMD(b *testing.B) {
	messages := getTestMessages()
	var pj *simdjson.ParsedJson
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, msg := range messages {
			_, _ = ParseMessageSIMD(msg, pj)
		}
	}
}

// benchmarkParseMessageSIMDWithReuse benchmarks the ParseMessageSIMD function with reusing the ParsedJson object
func BenchmarkParseMessageSIMDWithReuse(b *testing.B) {
	messages := getTestMessages()
	pj, _ := simdjson.Parse(make([]byte, 1024), nil) // initialize with some capacity
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, msg := range messages {
			_, _ = ParseMessageSIMD(msg, pj)
		}
	}
}

// getTestMessages returns a slice of test messages for benchmarking
func getTestMessages() [][]byte {
	// these are sample messages from the test cases
	return [][]byte{
		[]byte(`["EVENT","_",{"kind":1,"id":"dc90c95f09947507c1044e8f48bcf6350aa6bff1507dd4acfc755b9239b5c962","pubkey":"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d","created_at":1644271588,"tags":[],"content":"now that https://blueskyweb.org/blog/2-7-2022-overview was announced we can stop working on nostr?","sig":"230e9d8f0ddaf7eb70b5f7741ccfa37e87a455c9a469282e3464e2052d3192cd63a167e196e381ef9d7e69e9ea43af2443b839974dc85d8aaab9efe1d9296524"}]`),
		[]byte(`["EVENT",{"kind":1,"id":"dc90c95f09947507c1044e8f48bcf6350aa6bff1507dd4acfc755b9239b5c962","pubkey":"3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefa459d","created_at":1644271588,"tags":[],"content":"now that https://blueskyweb.org/blog/2-7-2022-overview was announced we can stop working on nostr?","sig":"230e9d8f0ddaf7eb70b5f7741ccfa37e87a455c9a469282e3464e2052d3192cd63a167e196e381ef9d7e69e9ea43af2443b839974dc85d8aaab9efe1d9296524"}]`),
		[]byte(`["AUTH","challenge-string"]`),
		[]byte(`["AUTH",{"kind":22242,"id":"9b86ca5d2a9b4aa09870e710438c2fd2fcdeca12a18b6f17ab9ebcdbc43f1d4a","pubkey":"79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798","created_at":1740505646,"tags":[["relay","ws://localhost:7777","2"],["challenge","3027526784722639360"]],"content":"","sig":"eceb827c4bba1de0ab8ee43f3e98df71194f5bdde0af27b5cda38e5c4338b5f63d31961acb5e3c119fd00ecef8b469867d060b697dbaa6ecee1906b483bc307d"}]`),
		[]byte(`["NOTICE","test notice message"]`),
		[]byte(`["EOSE","subscription123"]`),
		[]byte(`["CLOSE","subscription123"]`),
		[]byte(`["CLOSED","subscription123","reason: test closed"]`),
		[]byte(`["OK","3bf0c63fcb93463407af97a5e5ee64fa883d107ef9e558472c4eb9aaaefaaaaa",true,""]`),
		[]byte(`["COUNT","sub1",{"count":42}]`),
		[]byte(`["REQ","sub1",{"until":999999,"kinds":[1]}]`),
		[]byte(`["REQ","sub1z\\\"zzz",{"authors":["9b86ca5d2a9b4aa09870e710438c2fd2fcdeca12a18b6f17ab9ebcdbc43f1d4a","8eee10b2ce1162b040fdcfdadb4f888c64aaf87531dab28cc0c09fbdea1b663e","0deadebefb3c1a760f036952abf675076343dd8424efdeaa0f1d9803a359ed46"],"since":1740425099,"limit":2,"#x":["<","as"]},{"kinds":[2345,112],"#plic":["a"],"#ploc":["blblb","wuwuw"]}]`),
	}
}

// benchmarkParseMessagesFromFile benchmarks parsing messages from a file
// this function can be used if you have a file with messages
func BenchmarkParseMessagesFromFile(b *testing.B) {
	// read all messages into memory
	file, err := os.Open("testdata/messages.json")
	if err != nil {
		b.Fatal(err)
	}
	var messages [][]byte
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		messages = append(messages, []byte(scanner.Text()))
	}
	if err := scanner.Err(); err != nil {
		b.Fatal(err)
	}
	file.Close()

	// benchmark ParseMessage
	b.Run("ParseMessage", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, msg := range messages {
				_ = ParseMessage(msg)
			}
		}
	})

	// benchmark ParseMessageSIMD
	b.Run("ParseMessageSIMD", func(b *testing.B) {
		var pj *simdjson.ParsedJson
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, msg := range messages {
				_, _ = ParseMessageSIMD(msg, pj)
			}
		}
	})

	// benchmark ParseMessageSIMD with reuse
	b.Run("ParseMessageSIMDWithReuse", func(b *testing.B) {
		pj, _ := simdjson.Parse(make([]byte, 1024), nil) // initialize with some capacity
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, msg := range messages {
				_, _ = ParseMessageSIMD(msg, pj)
			}
		}
	})
}
