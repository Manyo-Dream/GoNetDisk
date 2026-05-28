package util

func CalcChunkCount(fileSize, chunkSize int64) int {
	count := int(fileSize / chunkSize)
	if fileSize%chunkSize != 0 {
		count++
	}
	return count
}
