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
	ScheduleTypeOneTime                   = "ONE_TIME"
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

func (j *Job) ReadyAndNext() (bool, *time.Time) {
	now := time.Now().UTC()
	switch j.Schedule {
	case ScheduleTypeNow:
		return true, nil
	case ScheduleTypeOneTime:
		return j.Marker.Before(now), nil
	case ScheduleTypeDaily:
		next := CalculateNextDaily(*j.Marker, *j.Next)
		return j.Next.Before(now), &next
	case ScheduleTypeWeekly:
		next := CalculateNextWeekly(*j.Marker, *j.Next)
		return j.Next.Before(now), &next
	case ScheduleTypeMonthly:
		next := CalculateNextMonthly(*j.Marker, *j.Next)
		return j.Next.Before(now), &next
	default:
		return false, nil
	}

}
func (j *Job) RunScan(results chan<- int) {
	time.Sleep(time.Second * 5)
	fmt.Println("finish scan")
	j.Running = false
	now := time.Now().UTC()
	j.LastEnd = &now
	fmt.Println("writing result")
	fmt.Printf("%+v", j)
}

func main() {
	// run right now
	job1 := NewJob(0, Template1.ID, ScheduleTypeNow, nil)

	// run once
	n := time.Now().Add(time.Second * 10)
	job2 := NewJob(1, Template1.ID, ScheduleTypeOneTime, &n)

	jobs := []Job{job1, job2}

	// ---------------

	tick := time.NewTicker(time.Second * 1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan struct{}, 1)
	results := make(chan int)
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
					fmt.Printf("job id = %d, marker = %s\n", job.ID, job.Marker)
					ready, next := job.ReadyAndNext()
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
	next := time.Date(
		curr.Year(),
		curr.Month(),
		curr.Day(),
		start.Hour(),
		start.Minute(),
		start.Second(),
		start.Nanosecond(),
		start.Location(),
	)
	return next
}

func CalculateNextWeekly(start time.Time, curr time.Time) time.Time {
	currPlusOneWeek := curr.AddDate(0, 0, 7)
	next := time.Date(
		currPlusOneWeek.Year(),
		currPlusOneWeek.Month(),
		currPlusOneWeek.Day(),
		start.Hour(),
		start.Minute(),
		start.Second(),
		start.Nanosecond(),
		start.Location(),
	)
	return next
}

func CalculateNextMonthly(start time.Time, curr time.Time) time.Time {
	nextMonth := curr.AddDate(0, 1, 0)
	daysInMonth := DaysIn(nextMonth.Month(), nextMonth.Year())
	startDay := start.Day()
	var nextDay int
	if startDay > daysInMonth {
		nextDay = daysInMonth
	} else {
		nextDay = startDay
	}
	next := time.Date(
		nextMonth.Year(),
		nextMonth.Month(),
		nextDay,
		start.Hour(),
		start.Minute(),
		start.Second(),
		start.Nanosecond(),
		start.Location(),
	)
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