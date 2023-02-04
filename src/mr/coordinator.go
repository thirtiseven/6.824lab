package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Coordinator struct {
	// Your definitions here.
	// map tasks
	mapTasks []Task
	// reduce tasks
	reduceTasks []Task
	// the number of map tasks
	nMapTasks int
	// the number of reduce tasks
	nReduceTasks int
	// the number of map tasks that have been completed
	nMapTasksCompleted int
	// the number of reduce tasks that have been completed
	nReduceTasksCompleted int
	// intermediate files
	intermediateFiles map[int][]string
	// the number of workers
	nWorkers int
	// workers list
	workers []WorkerInfo
	// given tasks for each worker
	givenTasks []Task
	// whether all map tasks have been completed
	allMapTasksCompleted bool
	// channel for map tasks
	mapTasksChan chan *Task
	// channel for reduce tasks
	reduceTasksChan chan *Task
	// mutex for the coordinator
	mu sync.Mutex
	// count for calling done
	doneCount int
}

type TaskStatus int

const (
	IDLE TaskStatus = iota
	IN_PROGRESS
	DONE
)

type TaskType int

const (
	MAP TaskType = iota
	REDUCE
	EMPTY
)

type Task struct {
	// The task id
	Id int
	// The task status
	Status TaskStatus
	// The task type: map or reduce
	Type TaskType
	// The file(s) to be processed
	Files []string
	// The number of reduce tasks
	NReduce int
}

type WorkerStatus int

const (
	WORKING WorkerStatus = iota
	FAILED
)

type WorkerInfo struct {
	// The worker id
	Id string
	// The worker status
	Status WorkerStatus
}

// Your code here -- RPC handlers for the worker to call.

func (c *Coordinator) RegisterWorker(args *WorkerArgs, reply *RegisterReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nWorkers++
	thisWorker := WorkerInfo{
		Id:     args.Id,
		Status: WORKING,
	}
	c.workers = append(c.workers, thisWorker)
	temp := RegisterReply{true}
	*reply = temp
	return nil
}

func (c *Coordinator) PingWorker(args *WorkerArgs, reply *RegisterReply) error {
	// ping the worker, if not responding in five seconds, mark it as failed
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, worker := range c.workers {
		if worker.Id == args.Id {
			c.workers[i].Status = WORKING
		}
	}
	temp := RegisterReply{true}
	*reply = temp
	return nil
}

func (c *Coordinator) GiveTask(args *WorkerArgs, reply *Task) error {
	if c.Done() {
		reply.Type = EMPTY
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.allMapTasksCompleted {
		if c.nMapTasksCompleted >= c.nMapTasks {
			flag := false
			for _, task := range c.givenTasks {
				if task.Type == MAP && task.Status != DONE {
					flag = true
				}
			}
			if !flag {
				c.allMapTasksCompleted = true
			}
			reply.Type = EMPTY
			return nil
		}
		if len(c.mapTasksChan) == 0 {
			// return a empty task
			// reply = &Task{}
			reply.Type = EMPTY
			return nil
		}
		maptask := <-c.mapTasksChan
		reply.Id = maptask.Id
		reply.Status = IN_PROGRESS
		reply.Type = MAP
		reply.Files = maptask.Files
		reply.NReduce = maptask.NReduce
		// if the task is not in the given tasks list, add it
		flag := false
		for i, task := range c.givenTasks {
			if task.Id == maptask.Id && task.Type == MAP {
				flag = true
				c.givenTasks[i].Status = IN_PROGRESS
				break
			}
		}
		if !flag {
			c.givenTasks = append(c.givenTasks, *reply)
		}

	} else {
		if len(c.reduceTasksChan) == 0 {
			// return a empty task
			// reply = &Task{}
			reply.Type = EMPTY
			return nil
		}
		reducetask := <-c.reduceTasksChan
		reply.Id = reducetask.Id
		reply.Status = IN_PROGRESS
		reply.Type = REDUCE
		reply.Files = c.intermediateFiles[reducetask.Id]
		reply.NReduce = reducetask.NReduce

		// if the task is not in the given tasks list, add it
		flag := false
		for i, task := range c.givenTasks {
			if task.Id == reply.Id && task.Type == REDUCE {
				flag = true
				c.givenTasks[i].Status = IN_PROGRESS
				break
			}
		}
		if !flag {
			c.givenTasks = append(c.givenTasks, *reply)
		}
	}
	// // check all map tasks have been completed
	// if c.nMapTasksCompleted >= c.nMapTasks {
	// 	c.allMapTasksCompleted = true
	// }
	// fmt.Printf("send task: %v\n", reply)
	return nil
}

func (c *Coordinator) CompleteTask(args *FinishedArgs, reply *FinishedReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	task := args.Task
	intermediateFiles := args.Filenames
	task.Status = DONE
	if task.Type == MAP {
		for _, file := range intermediateFiles {
			// filename format: mr-X-Y
			// split the filename by "-" and get the Y and convert it to int
			reduceId, _ := strconv.Atoi(strings.Split(file, "-")[2])
			c.intermediateFiles[reduceId] = append(c.intermediateFiles[reduceId], file)
			// fmt.Println("intermediate files: ", c.intermediateFiles)
		}
		c.nMapTasksCompleted++
	} else {
		c.nReduceTasksCompleted++
	}
	for i, t := range c.givenTasks {
		if t.Id == task.Id && t.Type == task.Type {
			// change the task status to done
			c.givenTasks[i].Status = DONE
			// fmt.Printf("task %v is done AGAIN\n", task)
			break
		}
	}
	// fmt.Printf("task %v is done\n", task)
	temp := FinishedReply{true}
	*reply = temp
	return nil
}

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	// println("nMapTasksCompleted", c.nMapTasksCompleted)
	ret := true
	if len(c.mapTasksChan) > 0 || len(c.reduceTasksChan) > 0 {
		ret = false
	}
	// check all tasks have been completed
	c.doneCount++
	for _, task := range c.givenTasks {
		if task.Status != DONE {
			if c.doneCount%100 == 0 {
				// fmt.Printf("task %v is not done\n", task)
			}
			ret = false
			// break
		}
	}
	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{
		nMapTasks:             len(files),
		nReduceTasks:          nReduce,
		nMapTasksCompleted:    0,
		nReduceTasksCompleted: 0,
		allMapTasksCompleted:  false,
		mapTasks:              make([]Task, len(files)),
		reduceTasks:           make([]Task, nReduce),
		mapTasksChan:          make(chan *Task, len(files)),
		reduceTasksChan:       make(chan *Task, nReduce),
		intermediateFiles:     make(map[int][]string),
		givenTasks:            make([]Task, 0),
		doneCount:             0,
	}

	// turn files into map tasks
	for i, file := range files {
		c.mapTasks[i] = Task{
			Id:      i,
			Status:  IDLE,
			Type:    MAP,
			Files:   []string{file},
			NReduce: nReduce,
		}
		// fmt.Printf("map task: %v\n", c.mapTasks[i])
		c.mapTasksChan <- &c.mapTasks[i]
	}
	// turn nReduce into reduce tasks
	for i := 0; i < nReduce; i++ {
		c.reduceTasks[i] = Task{
			Id:      i,
			Status:  IDLE,
			Type:    REDUCE,
			Files:   make([]string, 0),
			NReduce: nReduce,
		}
		c.reduceTasksChan <- &c.reduceTasks[i]
	}
	c.server()
	go func() {
		for {
			time.Sleep(10 * time.Second)
			// ping each worker to check if it is alive
			// if not, put the task back to the channel
			c.mu.Lock()
			// fmt.Println("given tasks: ", c.givenTasks)
			for i, task := range c.givenTasks {
				if task.Status == IN_PROGRESS {
					c.givenTasks[i].Status = IDLE
					// fmt.Println("task in progress: ", c.givenTasks[i])
					if c.givenTasks[i].Type == MAP {
						c.mapTasksChan <- &c.givenTasks[i]
					} else if c.givenTasks[i].Type == REDUCE {
						c.reduceTasksChan <- &c.givenTasks[i]
					}
				}
			}
			c.mu.Unlock()
		}
	}()

	return &c
}
