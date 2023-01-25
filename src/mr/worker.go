package mr

import (
	"fmt"
	"hash/fnv"
	"log"
	"net/rpc"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.
	// ask for a task from the coordinator and process it until
	// there is no more task to do (i.e. the coordinator returns
	// an empty task) or the coordinator is dead.

	for {
		task, ok := RequestTask()
		if !ok {
			break
		}
		if task.Type == MAP {
			// TODO process the map task
			ProcessMapTask(task, mapf)
		} else if task.Type == REDUCE {
			// TODO process the reduce task
			ProcessReduceTask(task, reducef)
		}
	}

	// uncomment to send the Example RPC to the coordinator.
	// CallExample()

}

func RequestTask() (Task, bool) {
	// send an RPC to the coordinator asking for a task
	args := WorkerStatus{}

	// TODO: fill in the worker status

	reply := Task{}

	ok := call("Coordinator.GivenTask", &args, &reply)
	if ok {
		return reply, true
	} else {
		fmt.Printf("call failed!\n")
		return Task{}, false
	}
}

func ProcessMapTask(task Task, mapf func(string, string) []KeyValue) {
	// TODO: process the map task
}

func ProcessReduceTask(task Task, reducef func(string, []string) string) {
	// TODO: process the reduce task
}

// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
