package lib

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"sort"
	"time"
)

func BytesToUint32(data []byte) (uint32, bool) {
	if len(data) != 4 {
		return 0, false
	}
	return binary.LittleEndian.Uint32(data), true
}

func Uin32ToBytes(number uint32) (ret_data []byte) {
	ret_data = make([]byte, 4)
	binary.LittleEndian.PutUint32(ret_data, number)
	return ret_data
}

func CalcPocketCount(length, pocket_length uint32) uint32 {
	return ((length / pocket_length) + 1)
}

//Function just for logging messages
func Log(message string) {
	fmt.Println(message)
}

func CombineBytesMap(data map[uint32][]byte) (ret_data []byte) {
	//Getting keys array
	var keys []int
	for k, _ := range data {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, k := range keys {
		ret_data = append(ret_data, data[uint32(k)]...)
	}
	return
}

func Random(min, max int) uint32 {
	rand.Seed(time.Now().UnixNano())
	return uint32(rand.Intn(max-min) + min)
}

//Used to complete UDP pocket length to be equal to UdpPocketLength
func FakeData(count int) []byte {
	ret_data := make([]byte, count)
	for i := 0; i < count; i++ {
		ret_data[i] = byte(15) //Just a random number to convert and set as a byte, just to be not empty :)
	}
	return ret_data
}
