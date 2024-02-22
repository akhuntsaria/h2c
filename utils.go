package main

func bytesStartWith(src []byte, search []byte) bool {
	if len(src) < len(search) {
		return false
	}

	for i := range search {
		if src[i] != search[i] {
			return false
		}
	}

	return true
}

// [0x00, 0xff, 0x01] -> 65281
func bytesToDec(bin []byte) int {
	dec := 0
	for i := 0; i < len(bin); i++ {
		dec |= int(bin[i]) << (8 * (len(bin) - i - 1))
	}
	return dec
}

func contains(haystack []string, needle string) bool {
	for _, str := range haystack {
		if str == needle {
			return true
		}
	}

	return false
}

// 65281, 2 -> [0xff, 0x01]
func decToBytes(dec int, len int) []byte {
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		bytes[i] = byte(dec >> (8 * (len - i - 1)))
	}
	return bytes
}
