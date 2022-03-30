#!/bin/bash

# if compiled to binary such as `lb-client`, you can change $COMMAND to `lb-client`.
COMMAND="go run ."

# successful requests
$COMMAND -url "www.ebay.com" -n 100

# invalid URL
$COMMAND -url "http:/:/bbb" -n 100

# domains not resolvable 
$COMMAND -url "aaa.bbb" -n 100

