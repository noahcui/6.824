# This is the Makefile helping you submit the labs.  
<br># Just create 6.824/api.key with your API key in it, 
<br># and submit your lab with the following command: 
<br>#     $ make [lab1|lab2a|lab2b|lab2c|lab3a|lab3b|lab4a|lab4b]
<br>
<br>LABS=" lab1 lab2a lab2b lab2c lab3a lab3b lab4a lab4b "
<br>
<br>%: check-%
<br>	@echo "Preparing $@-handin.tar.gz"
<br>	@if echo $(LABS) | grep -q " $@ " ; then \
<br>		echo "Tarring up your submission..." ; \
<br>		tar cvzf $@-handin.tar.gz \
<br>			"--exclude=src/main/pg-*.txt" \
<br>			"--exclude=src/main/diskvd" \
<br>			"--exclude=src/mapreduce/824-mrinput-*.txt" \
<br>			"--exclude=src/main/mr-*" \
<br>			"--exclude=mrtmp.*" \
<br>			"--exclude=src/main/diff.out" \
<br>			"--exclude=src/main/mrmaster" \
<br>			"--exclude=src/main/mrsequential" \
<br>			"--exclude=src/main/mrworker" \
<br>			"--exclude=*.so" \
<br>			Makefile src; \
<br>		if ! test -e api.key ; then \
<br>			echo "Missing $(PWD)/api.key. Please create the file with your key in it or submit the $@-handin.tar.gz via the web interface."; \
<br>		else \
<br>			echo "Are you sure you want to submit $@? Enter 'yes' to continue:"; \
<br>			read line; \
<br>			if test "$$line" != "yes" ; then echo "Giving up submission"; exit; fi; \
<br>			if test `stat -c "%s" "$@-handin.tar.gz" 2>/dev/null || stat -f "%z" "$@-handin.tar.gz"` -ge 20971520 ; then echo "File exceeds 20MB."; exit; fi; \
<br>			mv api.key api.key.fix ; \
<br>			cat api.key.fix | tr -d '\n' > api.key ; \
<br>			rm api.key.fix ; \
<br>			curl -F file=@$@-handin.tar.gz -F "key=<api.key" \
<br>			https://6824.scripts.mit.edu/2020/handin.py/upload > /dev/null || { \
<br>				echo ; \
<br>				echo "Submit seems to have failed."; \
<br>				echo "Please upload the tarball manually on the submission website."; } \
<br>		fi; \
<br>	else \
<br>		echo "Bad target $@. Usage: make [$(LABS)]"; \
<br>	fi
<br>
<br>.PHONY: check-%
<br>check-%:
<br>	@echo "Checking that your submission builds correctly..."
<br>	@./.check-build git://g.csail.mit.edu/6.824-golabs-2020 $(patsubst check-%,%,$@)
<br>