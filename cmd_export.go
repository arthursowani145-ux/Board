package main

import (
	"bufio"
	"fmt"
	"os"
)

func cmdExport() {
	p, err := myLogPath()
	if err != nil {
		fatal(err)
	}
	f, err := os.Open(p)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		fmt.Println(sc.Text())
	}
	if err := sc.Err(); err != nil {
		fatal(err)
	}
}
