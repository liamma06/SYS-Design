/*
	Task scheduler runs jobs at a specified time or in recurring intervals
used for background tasks like sending emails, generating reports, or cleaning up data.
*/

package main

import (
	"fmt"
	"sync"
	"time"
)

type Job struct {
	ID       string
	Interval time.Duration //how often job sohould run
	NextRun  time.Time
	Task     func()
}

type Scheduler struct {
	jobs map[string]*Job
	mu   sync.Mutex
	stop chan struct{} //channel to signal schedular to stop
}

// constructor for scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		jobs: make(map[string]*Job),
		stop: make(chan struct{}),
	}
}

func (s *Scheduler) Add(job *Job) {
	s.mu.Lock()
	job.NextRun = time.Now().Add(job.Interval) //set first run based on interval
	s.jobs[job.ID] = job                       //add job to scheduler's map
	s.mu.Unlock()
}

func (s *Scheduler) Remove(id string) {
	s.mu.Lock()
	delete(s.jobs, id) //remove job from scheduler's map
	s.mu.Unlock()
}

func (s *Scheduler) Start() {
	ticker := time.NewTicker(1 * time.Second) //check every second for due jobs

	defer ticker.Stop() //stop scheduler when Start returns

	for {
		select {
		case <-ticker.C: //check for due jobs every tick
			now := time.Now()
			s.mu.Lock()
			for _, job := range s.jobs {
				if now.After(job.NextRun) || now.Equal(job.NextRun) {
					go job.Task()                       //run job task in a separate goroutine
					job.NextRun = now.Add(job.Interval) //schedule next run based on interval
				}
			}
			s.mu.Unlock()
		case <-s.stop: //when stop signal received
			ticker.Stop() //stop the ticker
			return
		}
	}
}

func (s *Scheduler) Stop() {
	close(s.stop) //signal scheduler to stop, close channel broadcast to all listeners
}

func main() {
	s := NewScheduler()

	s.Add(&Job{
		ID:       "heartbeat",
		Interval: 2 * time.Second,
		Task: func() {
			fmt.Println("[heartbeat]  still alive —", time.Now().Format("15:04:05"))
		},
	})

	s.Add(&Job{
		ID:       "cleanup",
		Interval: 5 * time.Second,
		Task: func() {
			fmt.Println("[cleanup]    purging old records —", time.Now().Format("15:04:05"))
		},
	})

	s.Add(&Job{
		ID:       "report",
		Interval: 3 * time.Second,
		Task: func() {
			fmt.Println("[report]     generating summary —", time.Now().Format("15:04:05"))
		},
	})

	go s.Start()
	fmt.Println("scheduler started — running for 10 seconds")

	time.Sleep(10 * time.Second)
	s.Stop()
	fmt.Println("scheduler stopped")
}
