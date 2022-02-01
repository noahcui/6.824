#for repeating test, sometimes it can pass even if have bugs. 
#after a bunch of test, we can say that it is ok to say ok
i=0
s=0
f=0
max=0
min=2147483648
more=true
mm=true
log="logs/log$2.log"
fail="logs/fail$2.log"
runtime=`python -c 'import time; print int(time.time() * 1000)'`
# rm output$2 
rm $log
rm $fail
cmd="go test -log=err "$1" "$2
echo $cmd
while true
do
i=$[$i+1]

start=`python -c 'import time; print int(time.time() * 1000)'`
output=`$cmd`
end=`python -c 'import time; print int(time.time() * 1000)'`
if echo $output | grep -q "FAIL" ; then
#echo $i:
echo "Round: "$i" Fails: "$f"       FAILD"
printf "round number of failure: %d\n\n%s\n\n" $i "$output" >> $fail
f=$[$f+1]
else 
s=$[$s+1]
echo "Round: "$i" Fails: "$f
#echo $i:
fi

now=$(date)
rt=$[$end-$start]
att=$[($end-$runtime)/$i]
tt=$[($end-$runtime)/60]

if [[ $rt -gt $max ]] ; then
max=$rt
fi

if [[ $rt -lt $min ]] ; then
min=$rt
fi

if $more ; then
printf "%s | tT(mins): %3d.%-3d | " "$now" $[$tt/1000] $[$tt%1000] >> $log
fi

printf " f: %d | RT(s): %3d.%-3d | aRT(s): %3d.%-3d " $f  $[$rt/1000] $[$rt%1000] $[$att/1000] $[$att%1000] >> $log

if $mm ; then
printf " maxRT(s): %3d.%-3d | minRT(s): %3d.%-3d " $[$max/1000] $[$max%1000] $[$min/1000] $[$min%1000] >> $log
fi

printf "\n" >> $log
# rm output
done