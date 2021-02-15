# [Lab1: Map-Reduce(Distribute systems, Golang101)](https://pdos.csail.mit.edu/6.824/labs/lab-mr.html)


## Setting up GO
https://pdos.csail.mit.edu/6.824/labs/go.html

## Source Code

$ git clone git://g.csail.mit.edu/6.824-golabs-2021 6.824

## The Job
The job is to implement a distributed MapReduce, consisting of two programs, the coordinator and the worker. There will be just one coordinator process, and one or more worker processes executing in parallel. In a real system the workers would run on a bunch of different machines, but for this lab you'll run them all on a single machine. The workers will talk to the coordinator via RPC. Each worker process will ask the coordinator for a task, read the task's input from one or more files, execute the task, and write the task's output to one or more files. The coordinator should notice if a worker hasn't completed its task in a reasonable amount of time (for this lab, use ten seconds), and give the same task to a different worker.

## Test
#### How to test
```
  cd main
  ./test-mr.sh
```
#### Result
you'll see this if pass all tests
```
$ sh ./test-mr.sh
*** Starting wc test.
--- wc test: PASS
*** Starting indexer test.
--- indexer test: PASS
*** Starting map parallelism test.
--- map parallelism test: PASS
*** Starting reduce parallelism test.
--- reduce parallelism test: PASS
*** Starting crash test.
--- crash test: PASS
*** PASSED ALL TESTS
```
