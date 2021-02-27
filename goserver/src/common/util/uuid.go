package util

import (
	"encoding/base64"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

var escaper = strings.NewReplacer("9", "99", "-", "90", "_", "91")

func GenerateStandardUUID() string {
	return uuid.NewV4().String()
}

func GenerateUUID() string {
	return escaper.Replace(base64.RawURLEncoding.EncodeToString(uuid.NewV4().Bytes()))
}

func GenerateSessionId() string {
	return strconv.FormatInt(time.Now().Unix(), 36) +
		base64.RawURLEncoding.EncodeToString(uuid.NewV4().Bytes()) +
		strconv.FormatInt(int64(rand.Int31n(10)), 10)
}

func GenerateUUIDTest() {
	for i := 0; i < 100; i++ {
		fmt.Println(GenerateUUID())
		fmt.Println(GenerateSessionId())
	}
}
