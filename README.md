# taskpool

Introduction
------------

go 版任务池管理

Compatibility
-------------



Installation and usage
----------------------

The import path for the package is *github.com/maihb/taskpool*.

To install it, run:

    go get github.com/maihb/taskpool


License
-------




Example
-------

```Go
package main

import (
	"log"
	"time"

	"github.com/maihb/taskpool"
)

var msg_to_iot = taskpool.New("test111",
	taskpool.WithChNum(1),
	taskpool.WithChMaxWorker(3))

func main() {
	for i := 0; i < 1000; i++ {
		msg_to_iot.Push(&poolData{a: "1", b: i})
	}
	time.Sleep(1400 * time.Microsecond)
}

type poolData struct {
	a string
	b int
}

func (self *poolData) Do() {
	log.Printf("a=%s, b=%d\n", self.a, self.b)
	time.Sleep(1 * time.Second)
}
```

This example will generate the following output:

```
2022/07/13 17:23:16 test111 cN[1],cL[1024],iT[60],wN[3]. index[000] add worker[1] 
2022/07/13 17:23:16 a=1, b=0
2022/07/13 17:23:16 test111 cN[1],cL[1024],iT[60],wN[3]. index[000] add worker[2] 
2022/07/13 17:23:16 a=1, b=1
2022/07/13 17:23:16 test111 cN[1],cL[1024],iT[60],wN[3]. index[000] add worker[3] 
2022/07/13 17:23:16 a=1, b=2
2022/07/13 17:23:17 a=1, b=3
2022/07/13 17:23:17 a=1, b=4
2022/07/13 17:23:17 a=1, b=5
2022/07/13 17:23:18 a=1, b=6
2022/07/13 17:23:18 a=1, b=7
2022/07/13 17:23:18 a=1, b=8
```
