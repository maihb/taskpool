package taskpool

import (
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type ITask interface {
	Do()
}
type ITaskPool interface {
	Push(ITask) error
}

type opt func(*taskPool)

const (
	min_chan_num     = 1
	min_last_time    = 30 //最后一个协程的最小工作时间30秒
	CHANNEL_TOO_FULL = "channel too full"
)

type taskPool struct {
	//待初始化参数
	name        string //任务池名称，用来区分错误log
	chMaxWorker int    //单channel的最大协程数
	chNum       int    //channel数量
	chLen       int    //channel长度
	idleTime    int64  //多久没活跃则退出协程

	//运行时成员变量
	ch           []chan ITask
	chWorkerNum  []int
	chLastIncome []int64 //最后接收任务时间
	chMutex      []sync.RWMutex
	noFullDebug  bool
}

func WithChMaxWorker(num int) opt {
	return func(t *taskPool) {
		if num > 1 {
			t.chMaxWorker = num
		}
	}
}
func WithChNum(num int) opt {
	return func(t *taskPool) {
		if num > 1 {
			t.chNum = num
		}
	}
}
func WithChLen(l int) opt {
	return func(t *taskPool) {
		if l > 256 {
			t.chLen = l
		}
	}
}
func WithIdleTime(s int) opt {
	return func(t *taskPool) {
		if s > 10 {
			t.idleTime = int64(s)
		}
	}
}
func WithNoFullDebug() opt {
	return func(t *taskPool) {
		t.noFullDebug = true
	}
}

func New(name string, opts ...opt) ITaskPool {
	defaulttaskPool := &taskPool{
		name:        name,
		chMaxWorker: min_chan_num,
		chNum:       min_chan_num,
		chLen:       1024, //默认channel长度：1024
		idleTime:    60,   //默认60秒没工作，则退出
	}
	for _, o := range opts {
		o(defaulttaskPool)
	}

	for i := 0; i < defaulttaskPool.chNum; i++ {
		defaulttaskPool.ch = append(defaulttaskPool.ch, make(chan ITask, defaulttaskPool.chLen))
	}
	defaulttaskPool.chWorkerNum = make([]int, defaulttaskPool.chNum)
	defaulttaskPool.chLastIncome = make([]int64, defaulttaskPool.chNum)
	defaulttaskPool.chMutex = make([]sync.RWMutex, defaulttaskPool.chNum)
	return defaulttaskPool
}

func (t *taskPool) Push(task ITask) error {
	if t == nil || task == nil {
		return nil
	}
	index := rand.Uint32() % uint32(t.chNum)
	t.chMutex[index].RLock()
	num := t.chWorkerNum[index]
	if num < 2 {
		atomic.StoreInt64(&t.chLastIncome[index], time.Now().Unix())
	}
	t.chMutex[index].RUnlock()

	if num < min_chan_num {
		num++
		go t.worker(index)
	}

	select {
	case t.ch[index] <- task:
		//当前channel大于10分之一，增加worker
		if num < t.chMaxWorker && len(t.ch[index]) > t.chLen/10 {
			go t.worker(index)
		}
	default:
		if !t.noFullDebug {
			log.Printf("%s cN[%d],cL[%d],wN[%d] is too full\n",
				t.name, t.chNum, t.chLen, t.chMaxWorker)
		}
		return fmt.Errorf(CHANNEL_TOO_FULL)
	}
	return nil
}

func (t *taskPool) worker(index uint32) {
	t.chMutex[index].Lock()
	if t.chWorkerNum[index] >= t.chMaxWorker {
		t.chMutex[index].Unlock()
		return
	} else {
		t.chWorkerNum[index]++
		log.Printf("%s cN[%d],cL[%d],iT[%d],wN[%d]. index[%03d] add worker[%d] \n",
			t.name, t.chNum, t.chLen, t.idleTime,
			t.chMaxWorker, index, t.chWorkerNum[index])
		t.chMutex[index].Unlock()
	}

	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		if err := recover(); err != nil {
			var buf [4096]byte
			n := runtime.Stack(buf[:], false)
			log.Printf("TaskPool[%s] %v\n==> %s\n", t.name, err, string(buf[:n]))
			go t.worker(index)
		}
	}()
	lastJob := time.Now().Unix()
	for {
		select {
		case v := <-t.ch[index]:
			v.Do()
			lastJob = time.Now().Unix()
		case <-ticker.C:
			now := time.Now().Unix()
			if now-lastJob <= t.idleTime {
				continue
			}

			t.chMutex[index].Lock()
			//如果是最后一个协程，距离上次任务入盏时间一定时间内不退出
			if t.chWorkerNum[index] == 1 && now-atomic.LoadInt64(&t.chLastIncome[index]) < min_last_time {
				t.chMutex[index].Unlock()
				continue
			} else {
				log.Printf("%s cN[%d],cL[%d],iT[%d],wN[%d]. index[%03d] delete worker[%d] \n",
					t.name, t.chNum, t.chLen, t.idleTime,
					t.chMaxWorker, index, t.chWorkerNum[index])
				t.chWorkerNum[index]--
				t.chMutex[index].Unlock()
				return
			}

		}
	}
}
