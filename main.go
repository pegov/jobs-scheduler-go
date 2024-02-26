package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

var (
	ScheduleTypeNow                       = "NOW"
	ScheduleTypeOnce                      = "ONCE"
	ScheduleTypeDaily                     = "DAILY"
	ScheduleTypeWeekly                    = "WEEKLY"
	ScheduleTypeMonthly                   = "MONTHLY"
	ScheduleTypeSpecificDaysPerWeek       = "SPECIFIC_DAYS_PER_WEEK"      // TODO
	ScheduleTypeSpecificDaysPerMonth      = "SPECIFIC_DAYS_PER_MONTH"     // TODO
	ScheduleTypeSpecificDaysIndependently = "SPECIFIC_DAYS_INDEPENDENTLY" // TODO
)

type Template struct {
	ID       int
	Username string
	Password string
}

var (
	Template1 = Template{
		ID:       1,
		Username: "testuser1",
		Password: "hunter2",
	}
)

type Job struct {
	ID         int
	TemplateID int
	Schedule   string
	Marker     *time.Time
	Next       *time.Time
	LastStart  *time.Time
	LastEnd    *time.Time
	Running    bool
	Done       bool // no more planned jobs
}

type JobResult struct {
	ID int
}

func NewJob(id int, templateID int, schedule string, marker *time.Time) Job {
	// TODO: if marker before now, calculate next
	return Job{
		ID:         id,
		TemplateID: templateID,
		Schedule:   schedule,
		Marker:     marker,
		Next:       marker,
	}
}

func (j *Job) ReadyAndNext(now time.Time) (bool, *time.Time) {
	switch j.Schedule {
	case ScheduleTypeNow:
		return true, nil
	case ScheduleTypeOnce:
		return j.Marker.Before(now), nil
	case ScheduleTypeDaily:
		next := CalculateNextDaily(*j.Next, now)
		fmt.Printf("COMPARING: %s with %s\n", j.Next, now)
		return j.Next.Before(now), &next
	case ScheduleTypeWeekly:
		next := CalculateNextWeekly(*j.Next, now)
		fmt.Printf("COMPARING: %s with %s\n", j.Next, now)
		return j.Next.Before(now), &next
	case ScheduleTypeMonthly:
		next := CalculateNextMonthly(*j.Next, now, *j.Marker)
		fmt.Printf("COMPARING: %s with %s\n", j.Next, now)
		return j.Next.Before(now), &next
	default:
		return false, nil
	}

}
func (j *Job) RunScan(results chan<- int) {
	time.Sleep(time.Second * 5)
	fmt.Println("finish scan")
	j.Running = false
	now := utc()
	j.LastEnd = &now
	fmt.Println("writing result")
	fmt.Printf("%+v", j)
}

func main() {
	// t1 := utc().AddDate(0, 2, 4)
	// fmt.Printf("TEST NEXT DATE: %s\n", t1)

	// t2 := utc().AddDate(0, 3, 5)
	// fmt.Printf("TEST CURR DATE: %s\n", t2)

	// next := t1
	// for next.Before(t2) {
	// 	next = next.AddDate(0, 1, 0)
	// }

	// fmt.Printf("TEST NEXT2 DATE: %s\n", next)

	// ---------------

	// run right now
	job1 := NewJob(0, Template1.ID, ScheduleTypeNow, nil)

	// run once
	n1 := utc().Add(time.Second * 10)
	job2 := NewJob(1, Template1.ID, ScheduleTypeOnce, &n1)

	// run daily
	// n2 := utc().Add(time.Second * -10)
	n2 := utc().Add(time.Second * 10)
	job3 := NewJob(2, Template1.ID, ScheduleTypeDaily, &n2)

	// run once a week
	n3 := utc().Add(time.Second * 10)
	job4 := NewJob(3, Template1.ID, ScheduleTypeWeekly, &n3)

	// run once a month
	n4 := utc().AddDate(0, 0, 3)
	// n4 := utc().AddDate(0, 0, -7)
	job5 := NewJob(4, Template1.ID, ScheduleTypeMonthly, &n4)

	jobs := []Job{job1, job2, job3, job4, job5}

	// ---------------

	tick := time.NewTicker(time.Second * 1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan struct{}, 1)
	results := make(chan int)
	now := utc().AddDate(0, 0, 63).Add(time.Second * -1)
	fmt.Printf("NOW = %s\n", now)
	go func() {
		for {
			select {
			case t := <-tick.C:
				// ClearTerminal()
				fmt.Printf("tick %s\n", t)
				for i := range jobs {
					job := &jobs[i]
					if job.Done {
						continue
					}
					fmt.Printf("job id = %d, type = %s, marker = %s\n", job.ID, job.Schedule, job.Marker)
					ready, next := job.ReadyAndNext(now)
					fmt.Printf("ready = %v, next = %s\n", ready, next)
					if ready {
						if next != nil {
							fmt.Println("writing next")
							job.Next = next
						} else {
							fmt.Println("writing done")
							job.Done = true
						}
						if !job.Running {
							now := time.Now().UTC()
							job.LastStart = &now
							job.Running = true
							go job.RunScan(results)
						}
					}
				}
			case sig := <-sigs:
				fmt.Printf("sig %s\n", sig)
				done <- struct{}{}
			}
		}
	}()
	<-done
}

func CalculateNextDaily(start time.Time, curr time.Time) time.Time {
	next := start
	for next.Before(curr) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func CalculateNextWeekly(start time.Time, curr time.Time) time.Time {
	next := start
	for next.Before(curr) {
		next = next.AddDate(0, 0, 7)
	}
	return next
}

func CalculateNextMonthly(start time.Time, curr time.Time, marker time.Time) time.Time {
	var nextDay int

	next := start
	for next.Before(curr) {
		next = next.AddDate(0, 1, 0)
		daysInMonth := DaysIn(next.Month(), next.Year())
		if daysInMonth >= marker.Day() {
			nextDay = marker.Day()
		} else {
			nextDay = daysInMonth
		}
		next = time.Date(
			next.Year(),
			next.Month(),
			nextDay,
			start.Hour(),
			start.Minute(),
			start.Second(),
			start.Nanosecond(),
			start.Location(),
		)
	}
	return next
}

func DaysIn(m time.Month, year int) int {
	return time.Date(year, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func runCmd(name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func ClearTerminal() {
	switch runtime.GOOS {
	case "darwin":
		runCmd("clear")
	case "linux":
		runCmd("clear")
	case "windows":
		runCmd("cmd", "/c", "cls")
	default:
		runCmd("clear")
	}
}

func utc() time.Time {
	return time.Now().UTC()
}
