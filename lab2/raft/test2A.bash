#for repeating test, sometimes it can pass even if have bugs. 
#after a bunch of test, we can say that it is ok to say ok
i=0
s=0
f=0
runtime=`python -c 'import time; print int(time.time() * 1000)'`
while true
do
i=$[$i+1]
start=`python -c 'import time; print int(time.time() * 1000)'`
go test -run 2A -race > output
end=`python -c 'import time; print int(time.time() * 1000)'`
if grep -q "FAIL" output ; then
#echo $i:
cat output
f=$[$f+1]
else 
s=$[$s+1]
#echo $i:
fi
rt=$[$end-$start]
tt=$[$end-$runtime]
att=$[$tt/$i]
printf "round: %-10d |          fail/success: %d/%-10d |          round time(s): %d.%-10d |          ave run-time(s): %d.%-10d |          total run-time(s): %d.%-10d\n" $i $f $s  $[$rt/1000] $[$rt%1000] $[$att/1000] $[$att%1000] $[$tt/1000] $[$tt%1000]  
rm output
done