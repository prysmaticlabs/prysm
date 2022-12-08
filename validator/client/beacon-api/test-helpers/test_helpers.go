package test_helpers

func FillByteSlice(sliceLength int, value byte) []byte {
	bytes := make([]byte, sliceLength)

	for index := range bytes {
		bytes[index] = value
	}

	return bytes
}

func FillByteArraySlice(sliceLength int, value []byte) [][]byte {
	bytes := make([][]byte, sliceLength)

	for index := range bytes {
		bytes[index] = value
	}

	return bytes
}
