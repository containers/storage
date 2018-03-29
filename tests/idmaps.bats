#!/usr/bin/env bats

load helpers

@test "idmaps-create-apply-layer" {
	n=5
	host=2
	# Create some temporary files.
	for i in $(seq $n) ; do
		createrandom "$TESTDIR"/file$i
		chown ${i}:${i} "$TESTDIR"/file$i
		ln -s . $TESTDIR/subdir$i
	done
	# Use them to create some diffs.
	pushd $TESTDIR > /dev/null
	for i in $(seq $n) ; do
		tar cf diff${i}.tar subdir$i/
	done
	popd > /dev/null
	# Select some ID ranges.
	for i in $(seq $n) ; do
		uidrange[$i]=$((($RANDOM+32767)*65536))
		gidrange[$i]=$((($RANDOM+32767)*65536))
	done
	# Create a layer using the host's mappings.
	run storage --debug=false create-layer --hostuidmap --hostgidmap
	echo "$output"
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	lowerlayer="$output"
	# Mount the layer.
	run storage --debug=false mount $lowerlayer
	echo "$output"
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	lowermount="$output"
	# Copy the files in (host mapping, so it's fine), and set ownerships on them.
	cp -p "$TESTDIR"/file1 ${lowermount}
	cp -p "$TESTDIR"/file2 ${lowermount}
	cp -p "$TESTDIR"/file3 ${lowermount}
	cp -p "$TESTDIR"/file4 ${lowermount}
	cp -p "$TESTDIR"/file5 ${lowermount}
	# Create new layers.
	for i in $(seq $n) ; do
		if test $host -ne $i ; then
			run storage --debug=false create-layer --uidmap 0:${uidrange[$i]}:$(($n+1)) --gidmap 0:${gidrange[$i]}:$(($n+1)) $lowerlayer
		else
			uidrange[$host]=0
			gidrange[$host]=0
			run storage --debug=false create-layer --hostuidmap --hostgidmap $lowerlayer
		fi
		echo "$output"
		[ "$status" -eq 0 ]
		[ "$output" != "" ]
		upperlayer="$output"
		run storage --debug=false mount $upperlayer
		echo "$output"
		[ "$status" -eq 0 ]
		[ "$output" != "" ]
		uppermount="$output"
		run storage --debug=false apply-diff $upperlayer < ${TESTDIR}/diff${i}.tar
		echo "$output"
		[ "$status" -eq 0 ]
		[ "$output" = "" ]
		for j in $(seq $n) ; do
			# Check the inherited original files.
			cmp ${uppermount}/file$j "$TESTDIR"/file$j
			uid=$(stat -c %u ${uppermount}/file$j)
			gid=$(stat -c %g ${uppermount}/file$j)
			echo test found ${uid}:${gid} = expected $((${uidrange[$i]}+$j)):$((${gidrange[$i]}+$j)) for file$j
			test ${uid}:${gid} = $((${uidrange[$i]}+$j)):$((${gidrange[$i]}+$j))
			# Check the inherited/current layer's diff files.
			for k in $(seq $i) ; do
				cmp ${uppermount}/subdir$k/file$j "$TESTDIR"/file$j
				uid=$(stat -c %u ${uppermount}/subdir$k/file$j)
				gid=$(stat -c %g ${uppermount}/subdir$k/file$j)
				echo test found ${uid}:${gid} = expected $((${uidrange[$i]}+$j)):$((${gidrange[$i]}+$j)) for subdir$k/file$j
				test ${uid}:${gid} = $((${uidrange[$i]}+$j)):$((${gidrange[$i]}+$j))
			done
		done
		lowerlayer=$upperlayer
	done
}

@test "idmaps-create-diff-layer" {
	n=5
	host=2
	# Create some temporary files.
	for i in $(seq $n) ; do
		createrandom "$TESTDIR"/file$i
	done
	# Select some ID ranges.
	for i in 0 $(seq $n) ; do
		uidrange[$i]=$((($RANDOM+32767)*65536))
		gidrange[$i]=$((($RANDOM+32767)*65536))
	done
	# Create a layer using some random mappings.
	run storage --debug=false create-layer --uidmap 0:${uidrange[0]}:$(($n+1)) --gidmap 0:${gidrange[0]}:$(($n+1))
	echo "$output"
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	lowerlayer="$output"
	# Mount the layer.
	run storage --debug=false mount $lowerlayer
	echo "$output"
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	lowermount="$output"
	# Copy the files in, and set ownerships on them.
	for i in $(seq $n) ; do
		cp "$TESTDIR"/file$i ${lowermount}
		chown $((${uidrange[0]}+$i)):$((${gidrange[0]}+$i)) ${lowermount}/file$i
	done
	# Create new layers.
	for i in $(seq $n) ; do
		if test $host -ne $i ; then
			run storage --debug=false create-layer --uidmap 0:${uidrange[$i]}:$(($n+1)) --gidmap 0:${gidrange[$i]}:$(($n+1)) $lowerlayer
		else
			uidrange[$host]=0
			gidrange[$host]=0
			run storage --debug=false create-layer --hostuidmap --hostgidmap $lowerlayer
		fi
		echo "$output"
		[ "$status" -eq 0 ]
		[ "$output" != "" ]
		upperlayer="$output"
		run storage --debug=false mount $upperlayer
		echo "$output"
		[ "$status" -eq 0 ]
		[ "$output" != "" ]
		uppermount="$output"
		# Change the owner of one file to 0:0.
		chown ${uidrange[$i]}:${gidrange[$i]} ${uppermount}/file$i
		# Verify that the file is the only thing that shows up in a change list.
		run storage --debug=false changes $upperlayer
		echo "$output"
		[ "$status" -eq 0 ]
		[ "$output" != "" ]
		[ "${#lines[*]}" -eq 1 ]
		[ "$output" = "Modify \"/file$i\"" ]
		# Verify that the file is the only thing that shows up in a diff.
		run storage --debug=false diff -f $TESTDIR/diff.tar $upperlayer
		echo "$output"
		[ "$status" -eq 0 ]
		run tar tf $TESTDIR/diff.tar
		echo "$output"
		[ "$status" -eq 0 ]
		[ "$output" != "" ]
		[ "${#lines[*]}" -eq 1 ]
		[ "$output" = file$i ]
		# Check who owns that file, according to the diff.
		mkdir "$TESTDIR"/subdir$i
		pushd "$TESTDIR"/subdir$i > /dev/null
		run tar xf $TESTDIR/diff.tar
		[ "$status" -eq 0 ]
		run stat -c %u:%g file$i
		[ "$status" -eq 0 ]
		[ "$output" = 0:0 ]
		popd > /dev/null
		lowerlayer=$upperlayer
	done
}

@test "idmaps-create-container" {
	n=5
	host=2
	# Create some temporary files.
	for i in $(seq $n) ; do
		createrandom "$TESTDIR"/file$i
	done
	# Select some ID ranges.
	for i in 0 $(seq $(($n+1))) ; do
		uidrange[$i]=$((($RANDOM+32767)*65536))
		gidrange[$i]=$((($RANDOM+32767)*65536))
	done
	# Create a layer using some random mappings.
	run storage --debug=false create-layer --uidmap 0:${uidrange[0]}:$(($n+1)) --gidmap 0:${gidrange[0]}:$(($n+1))
	echo "$output"
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	lowerlayer="$output"
	# Mount the layer.
	run storage --debug=false mount $lowerlayer
	echo "$output"
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	lowermount="$output"
	# Copy the files in, and set ownerships on them.
	for i in $(seq $n) ; do
		cp "$TESTDIR"/file$i ${lowermount}
		chown $((${uidrange[0]}+$i)):$((${gidrange[0]}+$i)) ${lowermount}/file$i
	done
	# Create new layers.
	for i in $(seq $n) ; do
		if test $host -ne $i ; then
			run storage --debug=false create-layer --uidmap 0:${uidrange[$i]}:$(($n+1)) --gidmap 0:${gidrange[$i]}:$(($n+1)) $lowerlayer
		else
			uidrange[$host]=0
			gidrange[$host]=0
			run storage --debug=false create-layer --hostuidmap --hostgidmap $lowerlayer
		fi
		echo "$output"
		[ "$status" -eq 0 ]
		[ "$output" != "" ]
		upperlayer="$output"
		run storage --debug=false mount $upperlayer
		echo "$output"
		[ "$status" -eq 0 ]
		[ "$output" != "" ]
		uppermount="$output"
		# Change the owner of one file to 0:0.
		chown ${uidrange[$i]}:${gidrange[$i]} ${uppermount}/file$i
		# Verify that the file is the only thing that shows up in a change list.
		run storage --debug=false changes $upperlayer
		echo "$output"
		[ "$status" -eq 0 ]
		[ "$output" != "" ]
		[ "${#lines[*]}" -eq 1 ]
		[ "$output" = "Modify \"/file$i\"" ]
		lowerlayer=$upperlayer
	done
	# Create new containers based on the layer.
	imagename=idmappedimage
	storage create-image --name=$imagename $lowerlayer

	run storage --debug=false create-container --hostuidmap --hostgidmap $imagename
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	hostcontainer="$output"
	run storage --debug=false mount $hostcontainer
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	hostmount="$output"
	for i in $(seq $n) ; do
		run stat -c %u:%g "$hostmount"/file$i
		[ "$status" -eq 0 ]
		[ "$output" = 0:0 ]
	done

	run storage --debug=false create-container --hostuidmap --hostgidmap $imagename
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	hostcontainer="$output"
	run storage --debug=false mount $hostcontainer
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	hostmount="$output"
	for i in $(seq $n) ; do
		run stat -c %u:%g "$hostmount"/file$i
		[ "$status" -eq 0 ]
		[ "$output" = 0:0 ]
	done

	run storage --debug=false create-container --uidmap 0:${uidrange[$(($n+1))]}:$(($n+1)) --gidmap 0:${gidrange[$(($n+1))]}:$(($n+1)) $imagename
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	newmapcontainer="$output"
	run storage --debug=false mount $newmapcontainer
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	newmapmount="$output"
	for i in $(seq $n) ; do
		run stat -c %u:%g "$newmapmount"/file$i
		[ "$status" -eq 0 ]
		[ "$output" = ${uidrange[$(($n+1))]}:${gidrange[$(($n+1))]} ]
	done

	run storage --debug=false create-container $imagename
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	defmapcontainer="$output"
	run storage --debug=false mount $defmapcontainer
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	defmapmount="$output"
	for i in $(seq $n) ; do
		run stat -c %u:%g "$defmapmount"/file$i
		[ "$status" -eq 0 ]
		[ "$output" = ${uidrange[$n]}:${gidrange[$n]} ]
	done
}

@test "idmaps-parent-owners" {
	n=5
	# Create some temporary files.
	for i in $(seq $n) ; do
		createrandom "$TESTDIR"/file$i
	done
	# Select some ID ranges.
	uidrange=$((($RANDOM+32767)*65536))
	gidrange=$((($RANDOM+32767)*65536))
	# Create a layer using some random mappings.
	run storage --debug=false create-layer --uidmap 0:${uidrange}:$(($n+1)) --gidmap 0:${gidrange}:$(($n+1))
	echo "$output"
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	layer="$output"
	# Mount the layer.
	run storage mount $layer
	echo "$output"
	[ "$status" -eq 0 ]
	# Check who owns the parent directories.
	run storage --debug=false layer-parent-owners $layer
	echo "$output"
	[ "$status" -eq 0 ]
	# Assume that except for root and maybe us, there are no other owners of parent directories of our layer.
	if ! fgrep -q 'UIDs: [0]' <<< "$output" ; then
		fgrep -q 'UIDs: [0, '$(id -u)']' <<< "$output"
	fi
	if ! fgrep -q 'GIDs: [0]' <<< "$output" ; then
		fgrep -q 'GIDs: [0, '$(id -g)']' <<< "$output"
	fi
	# Create a new container based on the layer.
	imagename=idmappedimage
	storage create-image --name=$imagename $layer
	run storage --debug=false create-container $imagename
	[ "$status" -eq 0 ]
	[ "$output" != "" ]
	container="$output"
	# Mount the container.
	run storage mount $container
	echo "$output"
	[ "$status" -eq 0 ]
	# Check who owns the parent directories.
	run storage --debug=false container-parent-owners $container
	echo "$output"
	[ "$status" -eq 0 ]
	# Assume that except for root and maybe us, there are no other owners of parent directories of our container's layer.
	if ! fgrep -q 'UIDs: [0]' <<< "$output" ; then
		fgrep -q 'UIDs: [0, '$(id -u)']' <<< "$output"
	fi
	if ! fgrep -q 'GIDs: [0]' <<< "$output" ; then
		fgrep -q 'GIDs: [0, '$(id -g)']' <<< "$output"
	fi
}
