#for repeating test, sometimes it can pass even if have bugs. 
#after a bunch of test, we can say that it is ok to say ok
i=0
s=0
f=0
runtime=`python -c 'import time; print int(time.time() * 1000)'`
rm output 
rm logs/correct-fail 
rm logs/correct-log 
while true
do
i=$[$i+1]
start=`python -c 'import time; print int(time.time() * 1000)'`
output=`go test -run 3 -race`
end=`python -c 'import time; print int(time.time() * 1000)'`
if echo $output | grep -q "FAIL" ; then
#echo $i:
printf "round number of failure: %d\n\n%s\n\n" $i "$output" >> logs/correct-fail 
f=$[$f+1]
else 
s=$[$s+1]
#echo $i:
fi
now=$(date)
rt=$[$end-$start]
att=$[($end-$runtime)/$i]
tt=$[($end-$runtime)/60]
printf "round: %-10d |          fail/success: %d/%-10d |          round time(s): %2d.%-10d |          ave run-time(s): %2d.%-10d |           finish time: %s |          total run-time(mins): %d.%-10d\n" $i $f $s  $[$rt/1000] $[$rt%1000] $[$att/1000] $[$att%1000] "$now" $[$tt/1000] $[$tt%1000]  >> logs/correct-log 
# rm output
done