#for repeating test, sometimes it can pass even if have bugs. 
#after a bunch of test, we can say that it is ok to say ok
while true
date
do go test -run 2A -race | grep -E -o 'ok|FAIL'
done