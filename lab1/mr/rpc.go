package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"strconv"
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

// Add your RPC definitions here.
// Args:
type Args struct {
	Wid int
}

type Report struct {
	Jobtype int //1: map; 2: reduce
	Id      int
	Status  int //0: not assigned yet; 1: running 2: done
}

// Rely: the reply should be a filename of an as-yet-unstarted map task
type Reply struct {
	Filename string //the file name.
	Alldone  bool   //all jobs are done.
	Jobtype  int    //0 if nojob, 1 for map, 2 for reduce.
	Id       int    //the task id.
	Bucket   int    //the bucket id
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the master.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func masterSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
