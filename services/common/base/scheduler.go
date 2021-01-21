package base

import (
	"context"
	"sync"
	"time"

	"github.com/golang/glog"
)

// ID to state mapping
var schedulerStates sync.Map

// This stores the state of an ID
type schedulerState struct {
	JobQueue chan *jobWrapper
	Mutex    *sync.Mutex
}

type jobWrapper struct {
	Ctx   context.Context
	Job   func(context.Context)
	Delay time.Duration
}

// ScheduleJob schedules a job identified by the type ID that is executed after the delay measured after the last job.
// It dedupes the jobs to the maximum of the queue size for each job type.
func ScheduleJob(ctx context.Context, typeID string, job func(context.Context), delay time.Duration) bool {
	actual, ok := schedulerStates.LoadOrStore(typeID, &schedulerState{
		// Max pending of two. Reason is that
		// we do not want to lose processing an event while another is in progress.
		// Like in DB scan, after the scan has already started,
		// another event can change the already scanned DB record.
		JobQueue: make(chan *jobWrapper, 2),
		Mutex:    &sync.Mutex{},
	})
	state := actual.(*schedulerState)
	if !ok {
		// Start the goroutine on the first load. There are not many job types.
		go func() {
			for {
				currJob := <-state.JobQueue
				glog.Infof(PrefixRequestID(currJob.Ctx, "Picked up the next job type %s"), typeID)
				jobCtx, cancelFn := context.WithCancel(currJob.Ctx)
				timer := time.AfterFunc(currJob.Delay, func() {
					glog.Infof(PrefixRequestID(jobCtx, "Executing job type %s"), typeID)
					currJob.Job(jobCtx)
					glog.Infof(PrefixRequestID(jobCtx, "Executed job type %s"), typeID)
					cancelFn()
				})
				glog.Infof("Waiting for job type %s to complete", typeID)
				select {
				case <-ctx.Done():
					timer.Stop()
					schedulerStates.Delete(typeID)
					glog.Infof(PrefixRequestID(jobCtx, "Exiting scheduler for job type %s"), typeID)
					return
				case <-jobCtx.Done():
					glog.Infof(PrefixRequestID(jobCtx, "Wait completed for job type %s to complete"), typeID)
				}
			}
		}()
	}
	state.Mutex.Lock()
	defer state.Mutex.Unlock()
	if len(state.JobQueue) == cap(state.JobQueue) {
		glog.Infof(PrefixRequestID(ctx, "Maximum number (%d) of jobs already scheduled for job type %s"), cap(state.JobQueue), typeID)
		return false
	}
	state.JobQueue <- &jobWrapper{Ctx: ctx, Job: job, Delay: delay}
	return true
}
