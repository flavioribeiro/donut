# Moving player to static

## Date: 2/3/24
### Hypothesis: It's easy to do experimentation with local player
### Summary

Copy and adapt code from https://github.com/flavioribeiro/donut-video into static/demo/

# Investigating a potential memory leak

## Date: 2/2/24
### Hypothesis: There's a memory leak happening
### Signs: pprof/allocs
![allocs profile print screen 1](imgs/entry_memory_leak/allocs_pprof001.svg "allocs profile print screen 1")
### Summary

* Start the donut `make run`, 
* Check the general profiling http://localhost:6060/debug/pprof/?debug=1
* Keep refreshing the page, notice that the counters: heap, goroutine, threadcreate are stable
* Access the the demo http://localhost:8080/
* Go back to the profiling page http://localhost:6060/debug/pprof/?debug=1
* Notice that the prof counters are stable, regardless if you refresh multiple times
* Now click on `[Connect]`
* Observe how the perf counters go up http://localhost:6060/debug/pprof/?debug=1
* Keep refreshing the http://localhost:6060/debug/pprof/? and you'll see that the counters `goroutine` and `threacreate` **are stable** but `heap` and `allocs` **counters are growing**.
  * You can check the specific heap page as well http://localhost:6060/debug/pprof/heap?debug=1
  * You can run the command `go tool pprof http://localhost:6060/debug/pprof/heap` and type `web`, it'll generate a svg tree map of heap.
    * Snapshot 1 at time `Start + 4 m`
    * ![heap profile print screen 1](imgs/entry_memory_leak/pprof001.svg "heap profile print screen 1")
    * Snapshot 2 at time `Start + 15 m`
    * ![heap profile print screen 2](imgs/entry_memory_leak/pprof002.svg "heap profile print screen 2")
    * Snapshot 2 at time `Start + 8 h`
    * ![heap profile print screen 3](imgs/entry_memory_leak/pprof003.svg "heap profile print screen 3")
    * Docker stats `docker stats` at `Start`
    * ![docker stats print screen 1](imgs/entry_memory_leak/docker_stats1.png.webp "docker stats print screen 1")
    * Docker stats `docker stats` at `Start + 10m`
    * ![docker stats print screen 2](imgs/entry_memory_leak/docker_stats2.png.webp "docker stats print screen 2")

> ref: https://go101.org/article/memory-leaking.html
### Conclusion: there's not leak, I confused allocs growing nature.