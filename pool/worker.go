package pool

import (
	"log"
)

type Job struct {
	ID     int
	Input  string
	Output chan map[string]string
	Err    chan error
	Do     func(in string, outCh chan map[string]string, errCh chan error)
}

type Worker struct {
	ID            int
	WorkerChannel chan chan Job
	Channel       chan Job
	End           chan bool
}

// start worker
func (w *Worker) Start() {
	go func() {
		for {
			w.WorkerChannel <- w.Channel
			select {
			case job := <-w.Channel:
				job.Do(job.Input, job.Output, job.Err)
			case <-w.End:
				return
			}
		}
	}()
}

// end worker
func (w *Worker) Stop() {
	log.Printf("worker [%d] is stopping", w.ID)
	w.End <- true
}
