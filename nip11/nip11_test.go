package nip11

import "testing"

func TestAddSupportedNIP(t *testing.T) {
	info := RelayInformationDocument{}
	info.AddSupportedNIP(12)
	info.AddSupportedNIP(12)
	info.AddSupportedNIP(13)
	info.AddSupportedNIP(1)
	info.AddSupportedNIP(12)
	info.AddSupportedNIP(44)
	info.AddSupportedNIP(2)
	info.AddSupportedNIP(13)
	info.AddSupportedNIP(2)
	info.AddSupportedNIP(13)
	info.AddSupportedNIP(0)
	info.AddSupportedNIP(17)
	info.AddSupportedNIP(19)
	info.AddSupportedNIP(1)
	info.AddSupportedNIP(18)

	for i, v := range []int{0, 1, 2, 12, 13, 17, 18, 19, 44} {
		if info.SupportedNIPs[i] != v {
			t.Errorf("expected info.SupportedNIPs[%d] to equal %v, got %v",
				i, v, info.SupportedNIPs)
			return
		}
	}
}
