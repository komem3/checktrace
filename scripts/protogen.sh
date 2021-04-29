#!/bin/bash

dir=$(cd $(dirname $0); pwd)
cd $dir/../proto

buf generate
