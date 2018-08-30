#!/usr/bin/env bash
#
## package declarations
BUILD_NAME="aardvark"
RELEASE_VERSION="0.2"

## simple usage example
showUsage() {
	printf "Usage: $0 [-u|-d]
	-u    Update vendor directory via 'dep ensure -update' and build
	-d    Remove binary, lockfile, and vendor directory\n\n"
	exit 0
}

## install dep if needed
ensureDep() {
	which dep > /dev/null 2>&1
	if [ $? -ne 0 ]; then
		printf "Installing dep ... "
		go get -u github.com/golang/dep/cmd/dep
	fi
}

## exit toggle
should_exit=false

## read options
while getopts ":udr" opt; do
	case $opt in
		u)
			ensureDep
			printf "Updating vendor directory ... "
			dep ensure -update > /dev/null 2>&1
		;;
		d)
			printf "Removing binary, lockfile and vendor directory ... "
			rm -rf "${BUILD_NAME}" Gopkg.lock vendor
			printf "done.\n"
			should_exit=true
		;;
		*)
			showUsage
		;;
	esac
done

## remove options
shift $((OPTIND-1))

## exiting?
if [ $should_exit == true ]; then
	exit 0
fi

## ensure consistent vendor state
ensureDep && printf "Ensuring dependencies ..."
dep ensure > /dev/null 2>&1

## build binaries
printf "Building ... "

## build it
go build -o "${BUILD_NAME}" -ldflags="-s -w" > /dev/null

## go build return
RETVAL=$?

## check build status
if [ $RETVAL -ne 0 ]; then
	printf "\nError during build!\n"
	exit $RETVAL
fi

## all done
printf "done.\nUsage: ./${BUILD_NAME} -h\n"

## exit same as build
exit $RETVAL
