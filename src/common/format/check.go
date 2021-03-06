package format

import (
	"regexp"
)

//32长，数字、字母、下划线、横杠
func CheckAccount(s string) bool {
	ret, _ := regexp.MatchString(`^[\w-@\.]{3,32}$`, s)
	return ret
}

//32长，任意非空字符
func CheckPasswd(s string) bool {
	ret, _ := regexp.MatchString(`^\S{1,32}$`, s)
	return ret
}

//32长，任意非空字符，FIXME：脏字库排查
func CheckName(s string) bool {
	ret, _ := regexp.MatchString(`^\S{1,32}$`, s)
	return ret
}

func CheckValue(key, s string) (ret bool) {
	switch key {
	case "phone": //11位定长数字
		ret, _ = regexp.MatchString(`^[0-9]{11}$`, s)
	case "email": //通常邮箱地址
		ret, _ = regexp.MatchString(`^[\w-]{1,32}@[\w-]{1,32}\.[\w-]{2,4}$`, s)
	default:
		ret = false
	}
	return
}
