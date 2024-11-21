package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)


type Coordinator struct {
	// Your definitions here.
	mapTasks []Task
	reduceTasks []Task

	mapTaskStatus []TaskStatus
	reduceTaskStatus []TaskStatus

	nMap      int
	nReduce   int
	files     []string
	completed bool

	mu sync.Mutex // Protects shared state

	loggingEnabled bool // Enable logging for debugging
}

// Your code here -- RPC handlers for the worker to call.

func (c *Coordinator) AssignTask(args *struct{}, reply *Task) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Assign map tasks first
	for i, status := range c.mapTaskStatus {
		if !status.Completed && !status.Assigned {
			// Assign the task
			*reply = c.mapTasks[i]
			c.mapTaskStatus[i].Assigned = true
			c.mapTaskStatus[i].StartTime = time.Now()
			if c.loggingEnabled {
				log.Printf("Assigned map task: %v\n", reply)
			}
			return nil
		}
	}

	// If all map tasks are done, assign reduce tasks
	if c.allMapTasksDone() {
		for i, status := range c.reduceTaskStatus {
			if !status.Completed && !status.Assigned {
				// Assign the task
				*reply = c.reduceTasks[i]
				c.reduceTaskStatus[i].Assigned = true
				c.reduceTaskStatus[i].StartTime = time.Now()
				if c.loggingEnabled {
					log.Printf("Assigned reduce task: %v\n", reply)
				}
				return nil
			}
		}
	}

	// If all tasks are done, ask the worker to exit
	if c.allMapTasksDone() && c.allReduceTasksDone() {
		reply.TaskType = "exit"
		return nil
	}

	// No tasks available, ask the worker to wait
	reply.TaskType = "wait"
	if c.loggingEnabled {
		log.Println("No tasks available, worker should wait")
	}
	return nil
}

func (c *Coordinator) ReportTaskDone(args *Task, reply *struct{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if args.TaskType == "map" {
		// Mark map task as completed
		c.mapTaskStatus[args.ID].Completed = true
		c.mapTaskStatus[args.ID].Assigned = false
		if c.loggingEnabled {
			log.Printf("Map task %d completed\n", args.ID)
		}
	} else if args.TaskType == "reduce" {
		// Mark reduce task as completed
		c.reduceTaskStatus[args.ID].Completed = true
		c.reduceTaskStatus[args.ID].Assigned = false
		if c.loggingEnabled {
			log.Printf("Reduce task %d completed\n", args.ID)
		}
	}

	return nil
}

func (c *Coordinator) allMapTasksDone() bool {
	for _, status := range c.mapTaskStatus {
		if !status.Completed {
			return false
		}
	}
	return true
}

func (c *Coordinator) allReduceTasksDone() bool {
	for _, status := range c.reduceTaskStatus {
		if !status.Completed {
			return false
		}
	}
	return true
}

func (c *Coordinator) checkTaskTimeouts() {
	for {
		time.Sleep(time.Second)
		c.mu.Lock()

		// Check map tasks
		for i, status := range c.mapTaskStatus {
			if status.Assigned && !status.Completed && time.Since(status.StartTime) > 10*time.Second {
				// Reassign the task
				c.mapTaskStatus[i].Assigned = false
				if c.loggingEnabled {
					log.Printf("Reassigning map task %d due to timeout\n", i)
				}
			}
		}

		// Check reduce tasks
		for i, status := range c.reduceTaskStatus {
			if status.Assigned && !status.Completed && time.Since(status.StartTime) > 10*time.Second {
				// Reassign the task
				c.reduceTaskStatus[i].Assigned = false
				if c.loggingEnabled {
					log.Printf("Reassigning reduce task %d due to timeout\n", i)
				}
			}
		}

		c.mu.Unlock()
	}
}


//
// start a thread that listens for RPCs from worker.go
//
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

//
// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
//
func (c *Coordinator) Done() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if all reduce tasks are completed
	if c.allMapTasksDone() && c.allReduceTasksDone() {
		c.completed = true
		return true
	}

	return false
}

//
// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{
		files:     files,
		nMap:      len(files),
		nReduce:   nReduce,
		completed: false,
		loggingEnabled: false,
	}

	// Initialize task structures
	c.mapTasks = make([]Task, len(files))
	c.mapTaskStatus = make([]TaskStatus, len(files))
	c.reduceTasks = make([]Task, nReduce)
	c.reduceTaskStatus = make([]TaskStatus, nReduce)

	// Create map tasks
	for i, file := range files {
		c.mapTasks[i] = Task{
			ID:       i,
			File:     file,
			TaskType: "map",
			NumOfFiles:  nReduce,
		}
		c.mapTaskStatus[i] = TaskStatus{}
	}

	// Create reduce tasks
	for i := 0; i < nReduce; i++ {
		c.reduceTasks[i] = Task{
			ID:       i,
			TaskType: "reduce",
			NumOfFiles:  len(files),
		}
		c.reduceTaskStatus[i] = TaskStatus{}
	}

	// Start RPC server
	c.server()
	go c.checkTaskTimeouts()
	return &c
}
