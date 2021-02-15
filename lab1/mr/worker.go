package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

const timeout = 1

// for sorting by key.
type ByKey []KeyValue

func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

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
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.
	args := Args{}
	args.Wid = -1
	reply := Reply{}
	run := call("Master.RPCHandler", &args, &reply)
	if run == false {
		//fmt.Printf("something goes wrong\n")
	} else {
		//fmt.Printf("File name from server: %v\n", reply.Filename)
	}
	for run && !reply.Alldone {
		if reply.Jobtype == 1 {
			//job is mapping.
			////fmt.Printf("Mapping: %v\n", reply.Filename)
			//
			// read each input file,
			// pass it to Map,
			// accumulate the intermediate Map output.
			//

			// Open the file and store it in a buffer
			filename := reply.Filename
			file, err := os.Open(filename)
			////fmt.Printf("%s\n", filename)
			if err != nil {
				log.Fatalf("cannot open %v", filename)
			}
			content, err := ioutil.ReadAll(file)
			if err != nil {
				log.Fatalf("cannot read %v", filename)
			}
			file.Close()
			//pass it to Map???
			//copied form mrsequential.go, can't find where mapf is defined
			kva := mapf(filename, string(content))

			//

			intermediate := make(map[int]*os.File)
			for _, v := range kva {
				////fmt.Printf("fuck %v\n", i)
				id := ihash(v.Key) % reply.Bucket
				file, exist := intermediate[id]
				if !exist {

					file, err = ioutil.TempFile("", "unh")
					if err != nil {
						log.Fatal("error creating temp files %v", file)
					}

				}
				intermediate[id] = file
				enc := json.NewEncoder(file)
				err := enc.Encode(&v)
				if err != nil {
					log.Fatal("error writing %v\n", file)
				}
			}
			////fmt.Printf("\n\n\n\nout\n\n\n\n")
			tofind := "mr-" + strconv.Itoa(reply.Id) + "*"
			filelist, err := filepath.Glob(tofind)
			/*Not needed in a real distribute system.
			Because we are actually doing everything under one dir,
			may cause problems if this task comes from a "crashed" node
			*/
			for _, filename := range filelist {
				os.Remove(filename)
			}

			for key, val := range intermediate {
				oldname := val.Name()
				val.Close()
				newname := fmt.Sprintf("mr-%v-%v", reply.Id, key)
				/*Not needed in a real distribute system.
				Because we are actually doing everything under one dir,
				may cause problems if this task comes from a "crashed" node
				*/
				os.Remove(newname)
				err = os.Rename(oldname, newname)
				if err != nil {
					fmt.Printf("failed to xxx rename %v to %v\n", oldname, newname)
				}
			}
			report := Report{
				Jobtype: 1,
				Id:      reply.Id,
				Status:  2,
			}
			feedback := Report{}
			call("Master.JobReport", &report, &feedback)
		} else if reply.Jobtype == 2 {
			//job is reducing.
			////fmt.Printf("Reducing: %v\n", reply.Filename)
			//the tem file name
			tofind := "mr-*-" + strconv.Itoa(reply.Bucket)
			filelist, err := filepath.Glob(tofind)
			if err != nil {
				log.Fatal("cannot find %v\n", tofind)
			}
			intermediate := []KeyValue{}
			for _, filename := range filelist {
				file, err := os.Open(filename)
				if err != nil {
					log.Fatal("cannot open %v\n", filename)
				}
				dec := json.NewDecoder(file)
				for {
					var kv KeyValue
					if err := dec.Decode(&kv); err != nil {
						break
					}
					intermediate = append(intermediate, kv)
				}
				file.Close()
			}
			sort.Sort(ByKey(intermediate))
			oname := "mr-out-" + strconv.Itoa(reply.Bucket)
			/*Not needed in a real distribute system.
			Because we are actually doing everything under one dir,
			may cause problems if this task comes from a "crashed" node
			*/
			os.Remove(oname)
			ofile, _ := os.Create(oname)

			i := 0
			//fmt.Printf("%v\n", len(intermediate))
			for i < len(intermediate) {
				j := i + 1
				for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
					j++
				}
				values := []string{}
				for k := i; k < j; k++ {
					values = append(values, intermediate[k].Value)
				}
				output := reducef(intermediate[i].Key, values)

				// this is the correct format for each line of Reduce output.
				fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, output)

				i = j
			}

			ofile.Close()
			report := Report{
				Jobtype: 2,
				Id:      reply.Id,
				Status:  2,
			}
			feedback := Report{}
			call("Master.JobReport", &report, &feedback)
		} else {
			//no task
			//fmt.Printf("No task received, Jobtype %v\n", reply.Jobtype)
			time.Sleep(timeout * time.Second)
		}
		args = Args{}
		args.Wid = -1
		reply = Reply{}
		run = call("Master.RPCHandler", &args, &reply)
	}

	// uncomment to send the Example RPC to the master.
	// CallExample()

}

//
// example function to show how to make an RPC call to the master.
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
	call("Master.Example", &args, &reply)

	// reply.Y should be 100.
	//fmt.Printf("reply.Y %v\n", reply.Y)
}

//
// send an RPC request to the master, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := masterSock()
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
