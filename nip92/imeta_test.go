package nip92

import (
	"testing"

	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/require"
)

func TestIMetaParsing(t *testing.T) {
	for i, tcase := range []struct {
		expected IMeta
		tags     nostr.Tags
	}{
		{
			expected: nil,
			tags:     nostr.Tags{},
		},
		{
			expected: nil,
			tags:     nostr.Tags{{"t", "nothing"}},
		},
		{
			expected: nil,
			tags:     nostr.Tags{{"imeta", "nothing"}},
		},
		{
			expected: nil,
			tags: nostr.Tags{
				{
					"imeta",
					"url https://i.nostr.build/yhsyFkwxWlw7odSB.gif",
					"blurhash eDG*7p~AE34;E29x^ij?EMWBEMIWI;IpbI0g0gn+%1oyNGEMM|%1n%",
					"dim 225x191",
				},
				{
					"imeta",
					"blurhash qwkueh",
					"dim oxo",
				},
			},
		},
		{
			expected: IMeta{
				{
					URL:      "https://i.nostr.build/yhsyFkwxWlw7odSB.gif",
					Blurhash: "eDG*7p~AE34;E29x^ij?EMWBEMIWI;IpbI0g0gn+%1oyNGEMM|%1n%",
					Width:    225,
					Height:   191,
				},
			},
			tags: nostr.Tags{
				{
					"imeta",
					"url https://i.nostr.build/yhsyFkwxWlw7odSB.gif",
					"blurhash eDG*7p~AE34;E29x^ij?EMWBEMIWI;IpbI0g0gn+%1oyNGEMM|%1n%",
					"dim 225x191",
				},
			},
		},
		{
			expected: IMeta{
				{
					URL:      "https://image.nostr.build/94060676611fe7fca86588068d3f140607eda443f6d66d6b9754e93b0b8439ac.jpg",
					Blurhash: "eQF?RxZ}nNo~fk*0s6Z~kYem0iacsRkEsl9eW;oHkDoIxbtSobRkRi",
					Width:    3024,
					Height:   3024,
				},
				{
					URL:      "https://image.nostr.build/9bc3998f79c401bc9d3b3b74c1cbff0a3225f754194b94a6f5750ca6ea492846.jpg",
					Blurhash: "#2Ss1[RO%MIUx]R*-;WUtR0Kt6njof-;xu-pofxt4-M{yExt%2s.xaV[jF4TM|?bt6tmt6tRaybI0KtS=wRkWYWAtRjYt7tR%M?vNerWRjIBV@RjIAxYIpRP_3NGaKWXRj",
					Width:    1920,
					Height:   1494,
					Alt:      "Verifiable file url",
				},
			},
			tags: nostr.Tags{
				{
					"imeta",
					"url https://image.nostr.build/94060676611fe7fca86588068d3f140607eda443f6d66d6b9754e93b0b8439ac.jpg",
					"blurhash eQF?RxZ}nNo~fk*0s6Z~kYem0iacsRkEsl9eW;oHkDoIxbtSobRkRi",
					"dim 3024x3024",
				},
				{
					"imeta",
					"url https://image.nostr.build/9bc3998f79c401bc9d3b3b74c1cbff0a3225f754194b94a6f5750ca6ea492846.jpg",
					"m image/jpeg",
					"alt Verifiable file url",
					"x 80a8f087f6c45ec7fb9e8839e5af095df9b439a0836cdf70a244086b6b2c1a88",
					"size 121397",
					"dim 1920x1494",
					"blurhash #2Ss1[RO%MIUx]R*-;WUtR0Kt6njof-;xu-pofxt4-M{yExt%2s.xaV[jF4TM|?bt6tmt6tRaybI0KtS=wRkWYWAtRjYt7tR%M?vNerWRjIBV@RjIAxYIpRP_3NGaKWXRj",
					"ox 9bc3998f79c401bc9d3b3b74c1cbff0a3225f754194b94a6f5750ca6ea492846",
				},
			},
		},
	} {
		require.Equal(t, tcase.expected, ParseTags(tcase.tags), "case %d", i)
	}
}
