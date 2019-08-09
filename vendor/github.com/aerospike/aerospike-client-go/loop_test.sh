#!/bin/bash

set -e

ginkgo build -o test

for i in {1..50}
do
	./test -keepGoing -succinct -randomizeSuites -slowSpecThreshold=15 -focus "Scan|Query" -- -h vmu1804
done

