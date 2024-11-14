#!/bin/bash

SERVER_EXEC="build/http_server"
PROXY_EXEC="build/proxy"
PORT="8080"
PROXY_PORT="8081"
BASE_URL="http://localhost:$PORT"
PROXY_URL="http://localhost:$PROXY_PORT"
TEST_FILE="testfile.txt"
TEMP_RESPONSE_FILE="response.txt"
TEST_CONTENT="Test content"


cleanup() {
    kill $SERVER_PID $PROXY_PID
    rm -f "$TEST_FILE" "$TEMP_RESPONSE_FILE"
}


trap cleanup EXIT

$SERVER_EXEC $PORT &
SERVER_PID=$!
$PROXY_EXEC $PROXY_PORT &
PROXY_PID=$!

# Give the server a second to start
sleep 1

check_value() {
    local actual=$1
    local expected=$2
    local description=$3

    if [[ "$actual" == "$expected" ]]; then
        echo -e "\033[0;32m$description check passed (expected: '$expected')\033[0m"
    else
        echo -e "\033[0;31m$description check failed (expected: '$expected', got: '$actual')\033[0m"
        exit 1
    fi
}

# GET request for a non-existing file (should return 404)
echo "Running Test 1: GET non-existing file"
response=$(curl -o /dev/null -s -w "%{response_code}\n" -i -X GET "$BASE_URL/non_existent.html")
check_value "$response" "404" "Status code"

# POST request to create a file
echo "Running Test 2: POST to create a file"
echo "$TEST_CONTENT" > "$TEST_FILE"
response=$(curl -o /dev/null -s -w "%{response_code}\n" -i -X POST --data-binary "@$TEST_FILE" "$BASE_URL/$TEST_FILE")
check_value "$response" "201" "Status code"

# GET request to retrieve the created file
echo "Running Test 3: GET the created file"
response=$(curl -o "$TEMP_RESPONSE_FILE" -s -w "%{response_code}\n" -X GET "$BASE_URL/$TEST_FILE")
body=$(cat "$TEMP_RESPONSE_FILE")
check_value "$response" "200" "Status code"
check_value "$body" "$TEST_CONTENT" "Body content"

# GET request for an unsupported file type (should return 400)
echo "Running Test 4: GET unsupported file type"
response=$(curl -o /dev/null -s -w "%{response_code}\n" -i -X GET "$BASE_URL/unsupported.js")
check_value "$response" "400" "Status code"

# Unsupported HTTP method (should return 501)
echo "Running Test 5: Unsupported HTTP method"
response=$(curl -o /dev/null -s -w "%{response_code}\n" -i -X DELETE "$BASE_URL/$TEST_FILE")
check_value "$response" "501" "Status code"

# GET request for a non-existing file via proxy (should return 404)
echo "Running Test 6: GET non-existing file via proxy"
response=$(curl -o /dev/null -s -w "%{response_code}" -X GET "$PROXY_URL/non_existent.html" -x "$BASE_URL/non_existent.html")
check_value "$response" "404" "Proxy server status code for non-existing file"

# GET request to retrieve the created file via proxy
echo "Running Test 7: GET the created file via proxy"
response=$(curl -o "$TEMP_RESPONSE_FILE" -s -w "%{response_code}" -X GET "$PROXY_URL/$TEST_FILE" -x "$BASE_URL/$TEST_FILE")
body=$(cat "$TEMP_RESPONSE_FILE")
check_value "$response" "200" "Proxy server status code for retrieving file"
check_value "$body" "$TEST_CONTENT" "Proxy server body content for retrieved file"

# GET request for an unsupported file type via proxy (should return 400)
echo "Running Test 8: GET unsupported file type via proxy"
response=$(curl -o /dev/null -s -w "%{response_code}" -X GET "$PROXY_URL/unsupported.js" -x "$BASE_URL/unsupported.js")
check_value "$response" "400" "Proxy server status code for unsupported file type"

# Unsupported HTTP method via proxy (should return 501)
echo "Running Test 9: Unsupported HTTP method via proxy"
response=$(curl -o /dev/null -s -w "%{response_code}" -X DELETE "$PROXY_URL/$TEST_FILE" -x "$BASE_URL/$TEST_FILE")
check_value "$response" "501" "Proxy server status code for unsupported method"