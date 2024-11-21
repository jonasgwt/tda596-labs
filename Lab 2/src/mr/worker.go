package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net/rpc"
	"os"
	"time"
)

//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}


//
// main/mrworker.go calls this function.
//
func Worker(mapf func(string, string) []KeyValue, reducef func(string, []string) string) {
	for {
		// Request a task from the coordinator
		task := RequestTask()

		// Handle different task types
		switch task.TaskType {
		case "map":
			ExecuteMapTask(mapf, task)
		case "reduce":
			ExecuteReduceTask(reducef, task)
		case "wait":
			// Sleep briefly and then re-request a task
			time.Sleep(time.Second)
		case "exit":
			// Exit the worker gracefully
			return
		}
	}
}

func RequestTask() Task {
	args := struct{}{}
	reply := Task{}

	ok := call("Coordinator.AssignTask", &args, &reply)
	if !ok {
		log.Fatal("Failed to contact coordinator")
	}

	return reply
}

func ExecuteMapTask(mapf func(string, string) []KeyValue, task Task) {
	// Read the input file
	content, err := os.ReadFile(task.File)
	if err != nil {
		log.Fatalf("Cannot read file %v: %v", task.File, err)
	}

	// Call the map function
	kva := mapf(task.File, string(content))

	// Partition the key-value pairs into nReduce buckets
	intermediateFiles := make([]*os.File, task.NumOfFiles)
	encoders := make([]*json.Encoder, task.NumOfFiles)

	for i := 0; i < task.NumOfFiles; i++ {
		filename := fmt.Sprintf("mr-%d-%d", task.ID, i)
		file, err := os.Create(filename)
		if err != nil {
			log.Fatalf("Cannot create file %v: %v", filename, err)
		}
		intermediateFiles[i] = file
		encoders[i] = json.NewEncoder(file)
	}

	for _, kv := range kva {
		bucket := ihash(kv.Key) % task.NumOfFiles
		err := encoders[bucket].Encode(&kv)
		if err != nil {
			log.Fatalf("Cannot write to intermediate file: %v", err)
		}
	}

	// Close all intermediate files
	for _, file := range intermediateFiles {
		file.Close()
	}

	// Notify the coordinator that the task is done
	NotifyTaskDone(task)
}

func ExecuteReduceTask(reducef func(string, []string) string, task Task) {
	// Collect intermediate files for this reduce task
	intermediate := make(map[string][]string)

	for i := 0; i < task.NumOfFiles; i++ {
		filename := fmt.Sprintf("mr-%d-%d", i, task.ID)
		file, err := os.Open(filename)
		if err != nil {
			log.Fatalf("Cannot open file %v: %v", filename, err)
		}

		decoder := json.NewDecoder(file)
		for {
			var kv KeyValue
			if err := decoder.Decode(&kv); err != nil {
				break
			}
			intermediate[kv.Key] = append(intermediate[kv.Key], kv.Value)
		}
		file.Close()
	}

	// Perform the reduce operation
	outputFilename := fmt.Sprintf("mr-out-%d", task.ID)
	outputFile, err := os.Create(outputFilename)
	if err != nil {
		log.Fatalf("Cannot create output file %v: %v", outputFilename, err)
	}
	defer outputFile.Close()

	for key, values := range intermediate {
		result := reducef(key, values)
		fmt.Fprintf(outputFile, "%v %v\n", key, result)
	}

	// Notify the coordinator that the task is done
	NotifyTaskDone(task)
}

func NotifyTaskDone(task Task) {
	args := task
	reply := struct{}{}

	ok := call("Coordinator.ReportTaskDone", &args, &reply)
	if !ok {
		log.Fatalf("Failed to notify coordinator about task completion")
	}
}

//
// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
//
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

//
// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
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
