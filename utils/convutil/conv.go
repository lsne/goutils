package convutil

import "strconv"

func StringToUint16(s string) (uint16, error) {
	// ParseUint 的第三个参数指定 bitSize=16，表示目标类型是 uint16
	n, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, err
	}
	return uint16(n), nil
}
