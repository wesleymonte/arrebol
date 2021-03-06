package arrebol

import (
	"github.com/emanueljoivo/arrebol/storage"
	"log"
	"os"
	"strconv"
	"sync"
)

// no preemptive
type Scheduler struct {
	workers      []*Worker
	pendingTasks chan *storage.Task
	pendingPlans chan *AllocationPlan
	policy       Policy
	mutex        sync.Mutex
}

type Policy uint

const (
	Fifo Policy = iota
)

func (p Policy) String() string {
	return [...]string{"Fifo"}[p]
}

func (p Policy) schedule(plans chan *AllocationPlan) {
	switch p {
	case Fifo:
		for plan := range plans {
			go plan.execute()
		}
	default:
		log.Println("Just support fifo")
	}
}

func NewScheduler(policy Policy) *Scheduler {
	defaultWorkerPool, _ := strconv.Atoi(os.Getenv("STATIC_WORKER_POOL"))
	return &Scheduler{
		policy:       policy,
		workers:      make([]*Worker, defaultWorkerPool),
		pendingTasks: make(chan *storage.Task),
		pendingPlans: make(chan *AllocationPlan),
	}
}

func (s *Scheduler) Start() {
	// only support raw workers, for now, meaning jobs sent to the supervisor of this scheduler will run
	// uninsulated and on the Unix-type host operating system
	s.HireRawWorkers(Raw)
	go s.inferPlans()
	s.Schedule()
}

func (s *Scheduler) Schedule() {
	s.policy.schedule(s.pendingPlans)
}

// should be specific by node
func (s *Scheduler) HireRawWorkers(driver Driver) {
	switch driver {
	case Raw:
		log.Println("just support system level execution with static pool of workers")
		pool, _ := strconv.Atoi(os.Getenv("STATIC_WORKER_POOL"))

		for i := 0; i < pool; i++ {
			s.workers = append(s.workers, NewWorker(Raw))
		}

	case Docker:
		log.Println("not supported yet")
	default:
		log.Println("no worker type")
	}
}

func (s *Scheduler) AddTask(task *storage.Task) {
	s.pendingTasks <- task
}

type AllocationPlan struct {
	task *storage.Task
	worker *Worker
}

func (a *AllocationPlan) execute() {
	a.worker.Execute(a.task)

}

// Seeding to the channel of plans.
// Listening to the channel of pending tasks.
// Ever that a new task exists this method will be called
// generating a new resource allocation plan to execute the task
func (s *Scheduler) inferPlans() {
	for task := range s.pendingTasks {
		log.Println("new pending task")

		plan := s.inferPlanForTask(task)

		if plan != nil {
			s.pendingPlans <- plan // a channel is used here because only fifo's policy is supported
		} else {
			s.pendingTasks <- task
		}
	}
}

func (s *Scheduler) inferPlanForTask(task *storage.Task) *AllocationPlan {
	s.mutex.Lock()
	var w *Worker
	for _, worker := range s.workers {
		if worker.MatchAny(task) {
			w = worker
		}
	}
	defer s.mutex.Unlock()
	if w != nil {
		return	&AllocationPlan{
			task: task,
			worker: w,
		}
	} else {
		return nil
	}
}

