package mr

import (
	"errors"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
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
	intermediateFiles []string
	// the number of workers
	nWorkers int
	// whether all map tasks have been completed
	allMapTasksCompleted bool
	// channel for map tasks
	mapTasksChan chan *Task
	// channel for reduce tasks
	reduceTasksChan chan *Task
	// mutex for the coordinator
	mu sync.Mutex
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

// Your code here -- RPC handlers for the worker to call.

func (c *Coordinator) GiveTask(args *WorkerStatus, reply *Task) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Done() {
		// raise an error for no task to give
		return errors.New("no task to give")
	}
	if !c.allMapTasksCompleted {
		if c.nMapTasksCompleted == c.nMapTasks {
			// raise an error for all map tasks have been given
			// but not all map tasks have been completed
			return errors.New("all map tasks have been given")
		}
		reply = <-c.mapTasksChan
		reply.Status = IN_PROGRESS
	} else {
		reply = <-c.reduceTasksChan
		reply.Status = IN_PROGRESS
	}
	// check all map tasks have been completed
	if c.nMapTasksCompleted == c.nMapTasks {
		c.allMapTasksCompleted = true
	}
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
	ret := c.allMapTasksCompleted && len(c.reduceTasksChan) == 0

	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{
		mapTasks:              make([]Task, len(files)),
		reduceTasks:           make([]Task, nReduce),
		nMapTasks:             len(files),
		nReduceTasks:          nReduce,
		nMapTasksCompleted:    0,
		nReduceTasksCompleted: 0,
		intermediateFiles:     make([]string, 0),
		nWorkers:              0,
		allMapTasksCompleted:  false,
		mapTasksChan:          make(chan *Task, len(files)),
		reduceTasksChan:       make(chan *Task, nReduce),
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
	return &c
}
