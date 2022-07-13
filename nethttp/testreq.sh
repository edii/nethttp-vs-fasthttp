#!/bin/bash

sleep 7
echo POST / HTTP/1.1
echo Host: localhost
echo Content-Length: 5
echo
sleep 0.1 # Required in order to defeat buffering and flush headers in a separate packet
echo Dota2