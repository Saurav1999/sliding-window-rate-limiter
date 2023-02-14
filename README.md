# sliding-window-rate-limiter
Sliding window rate limiter implemented in Golang/ Go  using redis


Possible ways of Rate Limiting
1. Limit by API endpoint
2. Limit by IP
3. Limit by Custom user identifier like X-User-Id, Authorization token, etc.


## How to use?

Step 1: Create a config.json containing the configs to rate limit based on the type of rate limit in use.

the keys for the config to be used are as follows:
1. limitsAPI - when limit by API endpoint is in use.
2. limitByIp - when limit by IP
3. limitsUser - when limit by User identifier is in use.

Each of the key is optional. Use only config for those type that are in use. Need not add for the rest.
The content of config should look something like as demonstrated below. 
```sh
{
    
    "limitsAPI": [ 
      {
        "identifier": "http://localhost:5000/hello-api",
        "limit": 8,
        "window": 60,
        "unit": "seconds"
      },
      {
        "identifier": "http://localhost:5000/hello",
        "limit": 50,
        "window": 800,
        "unit": "seconds"
      }
    ],
    "limitsUser": {
      "limit": 4,
      "window":60,
      "unit": "seconds"
    },

    "limitsIp": {
      "limit": 20,
      "window": 1200,
      "unit": "seconds"
    }
  }
```



Step 2: Invoke Init method with config path and a bool config limitByDefaultOnFailure that controls the default behaviour in case rate limiter faces errors like issue in getting config or any exception and couldn't decide on whether to allow or reject the request based on the sliding window algorithm.
  limitByDefaultOnFailure: true // Reject/Limit the request in case of any failures by default.
  limitByDefaultOnFailure: false // Allow the request in case of any failures by default.
  
Step 3: Wrap request handler with RateLimiter function and specify LimitType and identifier.

LimitType can have following values:
1. LimitByUser
2. LimitByApi
3. LimitByIp
      
Identifier can have any unique string value as it gets used as key in redis but in case of LimitByIp and LimitByApi it is recommended to keep it as empty string("") due to following reasons.

1. LimitByIp - RemoteAddr from request is used as identifier by default
2. LimitByApi - Url(r.Host + r.URL.Path) is used as identifier and it must be present as a substring in the identifier in limitsAPI block of the config.
    
In case the identifier is not passed as empty string("") then that identifier should be present in config file in the identifier field in case of LimitByApi.


LimitByUser type requires special wrapping of a HandleFunc type to allow to calculate and pass custom unique field (Like X-User-Id, Authorization token, etc) from the request as shown in example below.
 
## Usage example:
```sh
//Sample handler that simply returns success response in case the request gets processed and not get dropped by rate limiter.
func hello(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusAccepted)
	resp := Response{
		Message:   "Hello there!",
		ErrorCode: 200,
	}
	payload, err := json.Marshal(resp)

	if err != nil {
		fmt.Println("Error marshaling")
	}
	w.Write([]byte(payload))
}


func main() {
	ratelimiter.Init("./config/config.json", true) // Pass config path and limitByDefaultOnFailure bool
	helloHandler := http.HandlerFunc(hello)
	helloHandlerWrapper := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { //Wrapper layer to allow passing of custom identifier from request
  
    //custom user identifier
    identifier:= r.Header.Get("X-User-ID")
		ratelimiter.RateLimiter(helloHandler, ratelimiter.LimitByUser, identifier ).ServeHTTP(w, r)
	})
	http.Handle("/hello", helloHandlerWrapper)
	http.Handle("/hello-api", ratelimiter.RateLimiter(helloHandler, ratelimiter.LimitByApi, ""))
	http.Handle("/hello-ip", ratelimiter.RateLimiter(helloHandler, ratelimiter.LimitByIp, ""))

	http.ListenAndServe(":5000", nil)

}
```
### To Learn more about how exactly to create the config file and use this library, switch to example branch in this Repository.
#### Feel free to create pull requests and add suggestions.
