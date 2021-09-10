package kafka

import "github.com/confluentinc/confluent-kafka-go/kafka"

func chunkSlice(slice []*kafka.Message, chunkSize int) [][]*kafka.Message {
	var chunks [][]*kafka.Message
	for {
		if len(slice) == 0 {
			break
		}
		if len(slice) < chunkSize {
			chunkSize = len(slice)
		}
		chunks = append(chunks, slice[0:chunkSize])
		slice = slice[chunkSize:]
	}
	return chunks
}
