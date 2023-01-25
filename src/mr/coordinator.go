package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
)

type Coordinator struct {
	// Your definitions here.

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

func (c *Coordinator) GivenTask(args *WorkerStatus, reply *Task) error {
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
	ret := false

	// Your code here.

	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}

	// Your code here.

	c.server()
	return &c
}
