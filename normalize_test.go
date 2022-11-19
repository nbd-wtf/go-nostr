package nostr

import "fmt"

func ExampleNormalizeURL() {
	fmt.Println(NormalizeURL(""))
	fmt.Println(NormalizeURL("wss://x.com/y"))
	fmt.Println(NormalizeURL("wss://x.com/y/"))
	fmt.Println(NormalizeURL("http://x.com/y"))
	fmt.Println(NormalizeURL(NormalizeURL("http://x.com/y")))
	fmt.Println(NormalizeURL("wss://x.com"))
	fmt.Println(NormalizeURL("wss://x.com/"))
	fmt.Println(NormalizeURL(NormalizeURL(NormalizeURL("wss://x.com/"))))
	fmt.Println(NormalizeURL("x.com"))
	fmt.Println(NormalizeURL("x.com/"))
	fmt.Println(NormalizeURL("x.com////"))
	fmt.Println(NormalizeURL("x.com/?x=23"))

	// Output:
	//
	// wss://x.com/y
	// wss://x.com/y
	// ws://x.com/y
	// ws://x.com/y
	// wss://x.com
	// wss://x.com
	// wss://x.com
	// wss://x.com
	// wss://x.com
	// wss://x.com
	// wss://x.com?x=23
}
