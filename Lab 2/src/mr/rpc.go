package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"strconv"
	"time"
)

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

type Task struct {
	ID        int
	File      string
	TaskType  string // "map" or "reduce"
	NumOfFiles   int    // Number of files to create or read from
}

type TaskStatus struct {
	Assigned   bool      // Whether the task is currently assigned
	WorkerID   int       // ID of the worker assigned to this task
	StartTime  time.Time // When the task was assigned
	Completed  bool      // Whether the task has been completed
}

// Add your RPC definitions here.


// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
