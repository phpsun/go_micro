package util

import (
	"bytes"
	"common/tlog"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func ContainInt64(ids []int64, id int64) bool {
	for _, i := range ids {
		if i == id {
			return true
		}
	}
	return false
}

func ContainInt32(ids []int32, id int32) bool {
	for _, i := range ids {
		if i == id {
			return true
		}
	}
	return false
}

func ContainString(arr []string, elem string) bool {
	for _, x := range arr {
		if elem == x {
			return true
		}
	}
	return false
}

// Convert 为了解决类似与 can't convert []int into []interface{}, 使用反射来解决
func ConvertToInterfaceArray(list interface{}) []interface{} {
	if reflect.TypeOf(list).Kind() != reflect.Slice {
		tlog.Error("Invalid Args: not a slice!")
		return nil
	}
	v := reflect.ValueOf(list)
	ret := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		ret[i] = v.Index(i).Interface()
	}
	return ret
}

func UniqueInt32Array(arr []int32) []int32 {
	count := len(arr)
	if count == 0 {
		return arr
	}
	retlen := 0
	tmpmap := make(map[int32]bool, count)
	for i := 0; i < count; i++ {
		item := arr[i]
		if _, ok := tmpmap[item]; !ok {
			tmpmap[item] = true
			arr[retlen] = item
			retlen++
		}
	}
	return arr[:retlen]
}

func UniqueInt64Array(arr []int64) []int64 {
	count := len(arr)
	if count == 0 {
		return arr
	}
	retlen := 0
	tmpmap := make(map[int64]bool, count)
	for i := 0; i < count; i++ {
		item := arr[i]
		if _, ok := tmpmap[item]; !ok {
			tmpmap[item] = true
			arr[retlen] = item
			retlen++
		}
	}
	return arr[:retlen]
}

func UniqueStringArray(arr []string) []string {
	count := len(arr)
	if count == 0 {
		return arr
	}
	retlen := 0
	tmpmap := make(map[string]bool, count)
	for i := 0; i < count; i++ {
		item := arr[i]
		if _, ok := tmpmap[item]; !ok {
			tmpmap[item] = true
			arr[retlen] = item
			retlen++
		}
	}
	return arr[:retlen]
}

func RandomInt64Array(vals []int64) []int64 {
	sz := len(vals)
	if sz == 0 {
		return vals
	}
	ret := make([]int64, sz)
	perm := rand.Perm(sz)
	for i := 0; i < sz; i++ {
		ret[i] = vals[perm[i]]
	}
	return ret
}

func TrimStringArray(arr []string) []string {
	count := len(arr)
	if count == 0 {
		return arr
	}
	retlen := 0
	for i := 0; i < count; i++ {
		item := arr[i]
		if len(item) > 0 {
			arr[retlen] = item
			retlen++
		}
	}
	return arr[:retlen]
}

func StringArrayToInt64Array(ss []string) []int64 {
	count := len(ss)
	ret := make([]int64, 0, count)
	for i := 0; i < count; i++ {
		if n, err := strconv.ParseInt(ss[i], 10, 64); err == nil {
			ret = append(ret, n)
		}
	}
	return ret
}

func Int64ArrayToString(arr []int64, delim string) string {
	var buffer strings.Builder
	count := len(arr)
	for i := 0; i < count; i++ {
		if i != 0 {
			buffer.WriteString(delim)
		}
		buffer.WriteString(strconv.FormatInt(arr[i], 10))
	}
	return buffer.String()
}

func Int32ArrayToString(arr []int32, delim string) string {
	var buffer strings.Builder
	count := len(arr)
	for i := 0; i < count; i++ {
		if i != 0 {
			buffer.WriteString(delim)
		}
		buffer.WriteString(strconv.FormatInt(int64(arr[i]), 10))
	}
	return buffer.String()
}

func StringToInt64Array(s string, delim string) []int64 {
	ss := strings.Split(s, delim)
	count := len(ss)
	ret := make([]int64, 0, count)
	for i := 0; i < count; i++ {
		if n, err := strconv.ParseInt(ss[i], 10, 64); err == nil {
			ret = append(ret, n)
		}
	}
	return ret
}

func StringToInt64(s string, defaultVal int64) int64 {
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	return defaultVal
}

func IntToBool(n int) bool {
	if n == 0 {
		return false
	}
	return true
}

func BoolToInt(n bool) int {
	if n {
		return 1
	}
	return 0
}

func HashCrc32(s string) uint32 {
	return crc32.ChecksumIEEE([]byte(s))
}

func HashCrc32WithInt64(n int64) uint32 {
	return crc32.ChecksumIEEE([]byte(strconv.FormatInt(n, 10)))
}

func GetMidnightUnixTime() int64 {
	now := time.Now()
	ltime := now.UTC()
	return (now.Unix() - int64(ltime.Hour()*3600+ltime.Minute()*60+ltime.Second())) + 86400
}

func GetWeek() int {
	week := int(time.Now().Unix() / (7 * 86400))
	return week
}

func ParseTime(s string) time.Time {
	if strings.Index(s, ",") >= 0 {
		t, _ := time.Parse(time.RFC1123, s)
		return t
	} else if strings.Index(s, ":") >= 0 {
		t, _ := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
		return t
	} else {
		t, _ := time.ParseInLocation("2006-01-02", s, time.Local)
		return t
	}
}

func FormatDate(t time.Time) string {
	y, m, d := t.Date()
	return fmt.Sprintf("%4d%02d%02d", y, m, d)
}

func FormatFullTime(t time.Time) string {
	y, m, d := t.Date()
	return fmt.Sprintf("%4d-%02d-%02d %02d:%02d:%02d", y, m, d, t.Hour(), t.Minute(), t.Second())
}

func GetIntranetIp() string {
	ifaces, _ := net.Interfaces()
	for _, itf := range ifaces {
		if strings.Index(itf.Name, "lo") == 0 { // loopback
			continue
		}
		addrs, _ := itf.Addrs()
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip.IsUnspecified() || ip.IsLoopback() || ip.To4() == nil {
				continue
			}
			return ip.String()
		}
	}
	return ""
}

func IpToInt(ip string) uint32 {
	var ret uint32
	binary.Read(bytes.NewBuffer(net.ParseIP(ip).To4()), binary.BigEndian, &ret)
	return ret
}

func GenServerId() int {
	return int(IpToInt(GetIntranetIp()) & 0xFFFF)
}

//case insensitive Contains
func CaseContains(s, substr string) bool {
	s, substr = strings.ToUpper(s), strings.ToUpper(substr)
	return strings.Contains(s, substr)
}

func EnsureDir(dir string) error {
	err := os.MkdirAll(dir, 0755)
	if err == nil || os.IsExist(err) {
		return nil
	} else {
		return err
	}
}

func CopyFile(src, dst string) error {
	from, err := os.Open(src)
	if err != nil {
		return err
	}
	defer from.Close()

	err = EnsureDir(filepath.Dir(dst))
	if err != nil {
		return err
	}

	to, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		return err
	}

	return nil
}

func IsFileExists(fname string) error {
	_, err := os.Stat(fname)
	return err
}

func GetDomainFromHost(host string) string {
	pos := strings.Index(host, ":")
	if pos >= 0 {
		host = host[:pos]
	}
	parts := strings.Split(host, ".")
	sz := len(parts)
	if sz <= 2 {
		return host
	}

	partsLen := 2
	part2 := parts[sz-2]
	switch part2 {
	case "com":
		fallthrough
	case "org":
		fallthrough
	case "net":
		fallthrough
	case "edu":
		fallthrough
	case "gov":
		partsLen = 3
	default:
		part1 := parts[sz-1]
		if part1 == "uk" || part1 == "jp" {
			switch part2 {
			case "co":
				fallthrough
			case "ac":
				fallthrough
			case "me":
				partsLen = 3
			}
		}
	}
	return strings.Join(parts[sz-partsLen:], ".")
}
