This is a loadbalanced HTTP client which distributes requests to multiple IPs.

## Design
The loadbalanced client has 3 modules: 
- client
- resolver
- selector

### Design: client
The client firstly parses URL and calls resolver to resolve IPs of the domain. If the URL is invalid or resolving fails, the program exits with error. Then, the client initializes the selector to select IPs and start a loop to send requests in parallel, and receives the results of each request from a channel. If number of successful requests reaches the specified number, the loop is done successfully. Each request first selects an IP from selector, then sent to the IP. If a request fails, the IP used for the request is marked as fail by the selector. The concurrency of requests is controlled by a rate limiting mechanism implemented using a channel. The channel has a fixed length of buffer, and will be written when a ticker times out. A request must be sent after a read from the channel. So that, the channel implements a token bucket rate limit algorithm.

### Design: resolver
The resolver returns available IPs for the specified host. The resolver can specify multiple DNS servers, lookup the host in all DNS servers in parallel and take the first successful result to use.

### Design: selector
The selector is an interface to select an availble IP and mark an IP fail or unavailable. The selector can use different LB algorithms to choose an IP. Here I implemented a round robin selector to return IPs in a round robin rule.

## Usage
`go run . -url www.ebay.com -n 100`

### params
- `-url`: URL to send requests to.
- `-n`: number of requests to send. Default value is 100.
- `-t`: duration of timeout of one request. Default value is 1s.
- `-dns`: DNS server used to resolve 