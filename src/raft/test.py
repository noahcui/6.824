#!/usr/bin/python

import threading
import time
import os
import sys

mu = threading.Lock()
success = 0
failed = 0
rounds = 0
totaltime = time.time() * 1000
A = 0.0
B = 0.0
C = 0.0
D = 0.0
begin = time.time()
pause = False
stop = False
rst = False
rr = 4
wt = 0


def SaveOldLogs(fail, log):
    if os.path.exists(fail):
        oldf = open(fail, "r", buffering=1)
        hisf = open("logs/history/"+fail[5:], "a", buffering=1)
        buff = oldf.read()
        hisf.write(buff)
        hisf.write("/n/n--------------------WoShiFenGeXian--------------------")
        oldf.close()
        hisf.close()
    if os.path.exists(log):
        oldl = open(log, "r", buffering=1)
        hisl = open("logs/history/"+log[5:], "a", buffering=1)
        buff = oldl.read()
        hisl.write(buff)
        hisl.write("/n/n--------------------WoShiFenGeXian--------------------")
        oldl.close()
        hisl.close()


def tester(threadnum, section):
    global mu
    global success
    global failed
    global rounds
    global fails
    global logs
    global totaltime
    global pause
    global D
    global stop
    global wt
    global rst
    time.sleep(threadnum * wt)
    threadName = "Worker-%d" % threadnum
    print("%s started\nYour cmd:" % (threadName))
    while 1:
        if pause:
            print("%s paused" % (threadName))
            time.sleep(10)
            continue
        if stop:
            print("% stopped" % (threadName))
            return
        if rst:
            print("counter rested at round %d" % rounds)
            failed = 0
            success = 0
            rst = False
        rounds += 1
        start = time.time() * 1000
        cmd = "go test -run %s -race" % section
        res = os.popen(cmd)
        output = res.read()
        end = time.time() * 1000
        mu.acquire()
        if "FAIL" in output:
            failed += 1
            out = "round: %d\n%s\n\n" % (rounds, output)
            fails.write(out)
        else:
            success += 1

        out = "round: % -10d | fail/success: % d/%-10d | round time(s): % 10f\n" % (
            rounds, failed, success, (end - start)/1000)
        logs.write(out)
        rate = float(success)/float(success + failed) * 100
        out = "%s: Total finished: %-7d|    success rate: %7f" % (
            section, success + failed, rate) + '%' + "    |    round time: %10f" % ((end - start)/1000)
        D = rate
        print(out)
        mu.release()


def kbd(threadName, delay):
    global pause
    global stop
    global rst
    while 1:
        time.sleep(1)
        str = raw_input("your cmd:")
        if str == 'p':
            pause = True
            print("will stop after this round")
        if str == 'r':
            pause = False
            print("start test")
        if str == "s":
            stop = True
            print("will stop after this rounds")
            return
        if str == 'n':
            print("will reset after this rounds")
            rst = True


section = "none"

if sys.argv[1] == "2A":
    section = "2A"
elif sys.argv[1] == "2B":
    section = "2B"
elif sys.argv[1] == "2C":
    section = "2C"
elif sys.argv[1] == "2D":
    section = "2D"
elif sys.argv[1] == "2":
    section = "2"
else:
    print("No this section %s\n" % sys.argv[1])
    exit()
if len(sys.argv) > 2:
    rr = int(sys.argv[2])
    print("#threads: %d\n" % rr)
if len(sys.argv) > 3:
    wt = int(sys.argv[3])
    print("wait before start a new worker: %d\n" % wt)
    if wt < 0:
        print("err")
        exit()
failname = "logs/fail-%s" % section
logname = "logs/log-%s" % section
SaveOldLogs(failname, logname)
fails = open(failname, "w", buffering=1)
logs = open(logname, "w", buffering=1)
threads = []
t = threading.Thread(target=kbd, args=("monitor", 2, ))
threads.append(t)
t.start()
for x in range(rr):
    name = "Thread-" + str(x)
    t = threading.Thread(target=tester, args=(x, section, ))
    threads.append(t)
    t.start()
for t in threads:
    t.join()
fails.close()
logs.close()

finish = time.time()
print("%f mins\n" % ((finish - begin)/60.0))
