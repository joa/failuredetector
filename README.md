[![Build Status](https://api.travis-ci.org/joa/failuredetector.svg)](http://travis-ci.org/joa/failuredetector) [![GoDoc](https://godoc.org/github.com/joa/failuredetector?status.svg)](http://godoc.org/github.com/joa/failuredetector)

This is a port of [Akka](https://akka.io)'s excellent implementation of the [PhiAccuralFailureDetector](https://github.com/akka/akka/blob/master/akka-remote/src/main/scala/akka/remote/PhiAccrualFailureDetector.scala) as 
proposed by [Hayashibara et al.](http://fubica.lsd.ufcg.edu.br/hp/cursos/cfsc/papers/hayashibara04theaccrual.pdf)

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
