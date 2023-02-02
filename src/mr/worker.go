package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"sort"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

type ByKey []KeyValue

func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

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

	CallExample()

	counter := 0
	for {
		// loop counter
		counter++
		// fmt.Printf("counter: %v\n", counter)
		task, ok := RequestTask()
		// fmt.Printf("task: %v\n", task)
		// fmt.Printf("ok: %v\n", ok)
		if !ok {
			break
		}
		if task.Type == MAP {
			filenames := ProcessMapTask(task, mapf)
			// fmt.Printf("filenames: %v\n", filenames)
			ReportCompleteTask(task, filenames)
		} else if task.Type == REDUCE {
			filename := ProcessReduceTask(task, reducef)
			ReportCompleteTask(task, []string{filename})
		}
	}

	// uncomment to send the Example RPC to the coordinator.
	// CallExample()

}

func ProcessMapTask(task Task, mapf func(string, string) []KeyValue) (filenames []string) {
	// fmt.Printf("task.Files: %v\n", task.Files)
	filename := task.Files[0]
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("cannot open %v", filename)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", filename)
	}
	file.Close()
	kva := mapf(filename, string(content))
	intermediateFiles := make([][]KeyValue, task.NReduce)
	for _, kv := range kva {
		i := ihash(kv.Key) % task.NReduce
		intermediateFiles[i] = append(intermediateFiles[i], kv)
	}
	filenames = make([]string, task.NReduce)
	for i := 0; i < task.NReduce; i++ {
		filename := fmt.Sprintf("mr-%d-%d", task.Id, i)
		filenames[i] = filename
		file, err := os.Create(filename)
		if err != nil {
			log.Fatalf("cannot create %v", filename)
		}
		enc := json.NewEncoder(file)
		for _, kv := range intermediateFiles[i] {
			err := enc.Encode(&kv)
			if err != nil {
				log.Fatalf("cannot encode %v", kv)
			}
		}
		file.Close()
	}
	return filenames
}

func ProcessReduceTask(task Task, reducef func(string, []string) string) (filename string) {
	kva := []KeyValue{}
	for _, filename := range task.Files {
		file, err := os.Open(filename)
		if err != nil {
			log.Fatalf("cannot open %v", filename)
		}
		dec := json.NewDecoder(file)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			kva = append(kva, kv)
		}
	}
	sort.Sort(ByKey(kva))
	oname := fmt.Sprintf("mr-out-%d", task.Id)
	ofile, _ := os.Create(oname)
	i := 0
	for i < len(kva) {
		j := i + 1
		for j < len(kva) && kva[j].Key == kva[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, kva[k].Value)
		}
		output := reducef(kva[i].Key, values)
		fmt.Fprintf(ofile, "%v %v\n", kva[i].Key, output)
		i = j
	}
	ofile.Close()
	return oname
}

func RequestTask() (Task, bool) {
	// send an RPC to the coordinator asking for a task
	args := WorkerStatus{}

	// TODO: fill in the worker status

	reply := Task{}

	ok := call("Coordinator.GiveTask", &args, &reply)

	if ok {
		// fmt.Printf("Received Task: %v\n", reply)
		return reply, true
	} else {
		fmt.Printf("call failed!\n")
		return Task{}, false
	}
}

func ReportCompleteTask(task Task, filenames []string) bool {
	// send an RPC to the coordinator to report that the task is complete

	args := FinishedArgs{
		Task:      task,
		Filenames: filenames,
	}

	reply := FinishedReply{}

	ok := call("Coordinator.CompleteTask", &args, &reply)

	if ok {
		// fmt.Printf("Task completed.\n")
		// fmt.Printf("Reply: %v\n", reply)
		return true
	} else {
		fmt.Printf("call failed!\n")
		return false
	}

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
		// fmt.Printf("reply.Y %v\n", reply.Y)
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
