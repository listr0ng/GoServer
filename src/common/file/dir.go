package file

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//dir, name := filepath.Dir(path), filepath.Base(path)

func GetExeDir() string {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	return dir + "/"
}
func IsDirExist(path string) bool {
	if fi, err := os.Stat(path); err != nil {
		return os.IsExist(err)
	} else {
		return fi.IsDir()
	}
}
func IsExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

//返回的文件名，都是相对exe的
//获取指定目录下的所有文件 --- names, err := filepath.Glob("csv/*.csv")
//获取指定目录及子目录下的所有文件，可以匹配后缀过滤 --- names, err := WalkDir("csv/", ".csv")
func WalkDir(dirPth, suffix string) ([]string, error) {
	ret := make([]string, 0, 16)
	err := filepath.Walk(dirPth, func(filename string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}
		if strings.HasSuffix(fi.Name(), suffix) {
			ret = append(ret, filename)
		}
		return nil
	})
	return ret, err
}

func ReadLine(filename string, cb func(string)) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	rd := bufio.NewReader(f)
	for {
		line, err := rd.ReadString('\n')
		if err != nil || io.EOF == err {
			break
		}
		cb(strings.TrimSpace(line))
	}
	return nil
}

func CreateFile(dir, name string, flag int) (*os.File, error) {
	if err := os.MkdirAll(dir, 0777); err != nil {
		return nil, err
	}
	if file, err := os.OpenFile(dir+name, flag|os.O_CREATE, 0666); err != nil {
		return nil, err
	} else {
		return file, nil
	}
}
