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

// Master handler
type Master struct {
	// Your definitions here.
	maptask    []Map
	reducetask []Reduce
	nreduce    int
	nfile      int
	mu         sync.Mutex
}

// Map task
type Map struct {
	id        int
	filename  string
	status    int //0: not assigned yet; 1: running 2: done
	starttime time.Time
}

// Reduce task
type Reduce struct {
	id        int
	bucket    int
	status    int //0: not assigned yet; 1: running 2: done
	starttime time.Time
}

func (m *Master) mapdone() bool {
	for _, v := range m.maptask {
		if v.status != 2 {
			return false
		}
	}
	return true
}

func (m *Master) reducedone() bool {
	for _, v := range m.reducetask {
		if v.status != 2 {
			return false
		}
	}
	return true
}

func (m *Master) Longtime() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, v := range m.maptask {
		if v.status == 1 {
			runtime := time.Since(v.starttime)
			if runtime.Seconds() > 10 {
				v.status = 0
				m.maptask[i] = v
			}
		}
	}

	for i, v := range m.reducetask {
		if v.status == 1 {
			runtime := time.Since(v.starttime)
			if runtime.Seconds() > 10 {
				v.status = 0
				m.reducetask[i] = v
			}
		}
	}

	return nil
}

// Your code here -- RPC handlers for the worker to call.
func (m *Master) RPCHandler(args *Args, reply *Reply) error {
	//check for long running task

	// first fininsh map tasks
	m.mu.Lock()
	defer m.mu.Unlock()
	//fmt.Printf("Received task reqirement\n")
	if !m.mapdone() {
		//fmt.Printf("Giving map tasks\n")
		for i, v := range m.maptask {
			if v.status == 0 {
				//reply package
				reply.Id = v.id
				reply.Filename = v.filename
				reply.Jobtype = 1
				reply.Bucket = m.nreduce
				reply.Alldone = false

				//Mark the task as running
				v.status = 1
				v.starttime = time.Now()
				m.maptask[i] = v
				return nil
			}
		}
		return nil
	}
	// then, do reduce jobs
	if !m.reducedone() {
		//fmt.Printf("Giving reduce tasks\n")
		for i, v := range m.reducetask {
			if v.status == 0 {
				//reply package
				reply.Id = v.id
				//reply.filename = v.bucket
				reply.Jobtype = 2
				reply.Alldone = false
				reply.Bucket = v.bucket

				//Mark the task as running
				v.status = 1
				v.starttime = time.Now()
				m.reducetask[i] = v
				return nil
			}
		}
		return nil
	}
	//all jobs are done
	if m.Done() {
		reply.Alldone = true
	}

	//fmt.Printf("No more task for now\n")
	return nil
}

func (m *Master) JobReport(report *Report, feedback *Report) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if report.Jobtype == 1 {
		m.maptask[report.Id].status = report.Status
	} else if report.Jobtype == 2 {
		m.reducetask[report.Id].status = report.Status
	} else {
		log.Fatalln("error!")
	}
	return nil
}

//
// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
//
func (m *Master) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

//
// start a thread that listens for RPCs from worker.go
//
func (m *Master) server() {
	rpc.Register(m)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := masterSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

//
// main/mrmaster.go calls Done() periodically to find out
// if the entire job has finished.
//
func (m *Master) Done() bool {

	ret := false

	// Your code here.
	if m.mapdone() {
		if m.reducedone() {
			ret = true
		}
	}
	return ret
}

//
// create a Master.
// main/mrmaster.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeMaster(files []string, nReduce int) *Master {
	m := Master{}

	// Your code here.
	m.maptask = make([]Map, len(files))
	for i, v := range files {
		m.maptask[i] = Map{
			id:       i,
			filename: v,
			status:   0,
		}
	}
	m.reducetask = make([]Reduce, nReduce)
	for i := 0; i < nReduce; i++ {
		m.reducetask[i] = Reduce{
			id:     i,
			bucket: i,
			status: 0,
		}
	}
	m.nreduce = nReduce
	m.nfile = len(files)
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			<-ticker.C
			m.Longtime()
		}
	}()
	m.server()
	return &m
}
