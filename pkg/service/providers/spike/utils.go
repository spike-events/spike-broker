package spike

//func chunkSlice(slice []*client.Message, chunkSize int) [][]*client.Message {
//	var chunks [][]*client.Message
//	for {
//		if len(slice) == 0 {
//			break
//		}
//		if len(slice) < chunkSize {
//			chunkSize = len(slice)
//		}
//		chunks = append(chunks, slice[0:chunkSize])
//		slice = slice[chunkSize:]
//	}
//	return chunks
//}
