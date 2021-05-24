#!/bin/bash

build_dir=$1
dest_dir=$2

# Copy debian files from mod_specific directory
cp ${build_dir}/packaging/debian/* ${dest_dir}/DEBIAN/

cd ${build_dir}
go build -o mission-engine || exit
mkdir -p ${dest_dir}/usr/bin
cp -f mission-engine ${dest_dir}/usr/bin/ && go clean || exit
