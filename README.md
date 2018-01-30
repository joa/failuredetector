Port of Akka's excellent implementation of the [PhiAccuralFailureDetector](https://github.com/akka/akka/blob/master/akka-remote/src/main/scala/akka/remote/PhiAccrualFailureDetector.scala) as 
proposed by [Hayashibara et al.](http://www.jaist.ac.jp/~defago/files/pdf/IS_RR_2004_010.pdf)

### Usage
```go
fd, err := failuredetector.New(
		8.0,
		200,
		500*time.Millisecond,
		0,
		500*time.Millisecond,
		nil)

if err != nil {
  // invalid arguments given (e.g. negative sample size)
  panic(err)
}

go func() {
  for {
    time.Sleep(1 * time.Second) // some expensive check
    fd.Heartbeat()
  }
}()

// health check
fmt.Println(fd.IsAvailable())
```
