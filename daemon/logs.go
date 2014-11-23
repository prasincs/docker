package daemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/daemon/graphdriver"
	"github.com/docker/docker/engine"
	"github.com/docker/docker/pkg/jsonlog"
	"github.com/docker/docker/pkg/tailfile"
	"github.com/docker/docker/pkg/timeutils"
)

func FindRootDirFromDriver(driver graphdriver.Driver) string {
	rootDir := ""
	statusArray := driver.Status()
	for i := range statusArray {
		name := statusArray[i][0]
		value := statusArray[i][1]
		if name == "Root Dir" {
			rootDir = value
			break
		}
	}
	return rootDir
}

func (daemon *Daemon) ContainerFlogs(job *engine.Job) engine.Status {
	if len(job.Args) != 1 {
		return job.Errorf("Usage: %s CONTAINER\n", job.Name)
	}
	log.Errorf("%v", job)
	var (
		name    = job.Args[0]
		logName = job.Getenv("logname")
	)

	rootDir := FindRootDirFromDriver(daemon.GraphDriver())

	log.Errorf("Root Dir %v", rootDir)
	log.Errorf("%v", logName)
	//if logName == "" {
	//	return job.Errorf("You need to give an identifier to read from")
	//}

	//logPath := "/var/log/bootstrap.log"

	// if tail == "" {
	// 	tail = "all"
	// }
	container := daemon.Get(name)
	if container == nil {
		return job.Errorf("No such container: %s", name)
	}
	//configResPath, _ := container.getRootResourcePath("config.json")
	arbitraryResPath := path.Join(container.RootfsPath(), logName)

	log.Errorf("%v", arbitraryResPath)

	log.Errorf("%v", container.RootfsPath())

	if file, err := os.Open(arbitraryResPath); err != nil {
		log.Errorf("Error opening file path %s : %v", arbitraryResPath, err)
	} else {
		if _, err := io.Copy(job.Stdout, file); err != nil {
			log.Errorf("Error streaming logs (stdout): %s", err)
		}
	}

	return engine.StatusOK

}

func (daemon *Daemon) ContainerLogs(job *engine.Job) engine.Status {
	if len(job.Args) != 1 {
		return job.Errorf("Usage: %s CONTAINER\n", job.Name)
	}

	var (
		name   = job.Args[0]
		stdout = job.GetenvBool("stdout")
		stderr = job.GetenvBool("stderr")
		tail   = job.Getenv("tail")
		follow = job.GetenvBool("follow")
		times  = job.GetenvBool("timestamps")
		lines  = -1
		format string
	)
	if !(stdout || stderr) {
		return job.Errorf("You must choose at least one stream")
	}
	if times {
		format = timeutils.RFC3339NanoFixed
	}
	if tail == "" {
		tail = "all"
	}
	container := daemon.Get(name)
	if container == nil {
		return job.Errorf("No such container: %s", name)
	}
	cLog, err := container.ReadLog("json")
	if err != nil && os.IsNotExist(err) {
		// Legacy logs
		log.Debugf("Old logs format")
		if stdout {
			cLog, err := container.ReadLog("stdout")
			if err != nil {
				log.Errorf("Error reading logs (stdout): %s", err)
			} else if _, err := io.Copy(job.Stdout, cLog); err != nil {
				log.Errorf("Error streaming logs (stdout): %s", err)
			}
		}
		if stderr {
			cLog, err := container.ReadLog("stderr")
			if err != nil {
				log.Errorf("Error reading logs (stderr): %s", err)
			} else if _, err := io.Copy(job.Stderr, cLog); err != nil {
				log.Errorf("Error streaming logs (stderr): %s", err)
			}
		}
	} else if err != nil {
		log.Errorf("Error reading logs (json): %s", err)
	} else {
		if tail != "all" {
			var err error
			lines, err = strconv.Atoi(tail)
			if err != nil {
				log.Errorf("Failed to parse tail %s, error: %v, show all logs", tail, err)
				lines = -1
			}
		}
		if lines != 0 {
			if lines > 0 {
				f := cLog.(*os.File)
				ls, err := tailfile.TailFile(f, lines)
				if err != nil {
					return job.Error(err)
				}
				tmp := bytes.NewBuffer([]byte{})
				for _, l := range ls {
					fmt.Fprintf(tmp, "%s\n", l)
				}
				cLog = tmp
			}
			dec := json.NewDecoder(cLog)
			l := &jsonlog.JSONLog{}
			for {
				if err := dec.Decode(l); err == io.EOF {
					break
				} else if err != nil {
					log.Errorf("Error streaming logs: %s", err)
					break
				}
				logLine := l.Log
				if times {
					logLine = fmt.Sprintf("%s %s", l.Created.Format(format), logLine)
				}
				if l.Stream == "stdout" && stdout {
					io.WriteString(job.Stdout, logLine)
				}
				if l.Stream == "stderr" && stderr {
					io.WriteString(job.Stderr, logLine)
				}
				l.Reset()
			}
		}
	}
	if follow && container.IsRunning() {
		errors := make(chan error, 2)
		wg := sync.WaitGroup{}

		if stdout {
			wg.Add(1)
			stdoutPipe := container.StdoutLogPipe()
			defer stdoutPipe.Close()
			go func() {
				errors <- jsonlog.WriteLog(stdoutPipe, job.Stdout, format)
				wg.Done()
			}()
		}
		if stderr {
			wg.Add(1)
			stderrPipe := container.StderrLogPipe()
			defer stderrPipe.Close()
			go func() {
				errors <- jsonlog.WriteLog(stderrPipe, job.Stderr, format)
				wg.Done()
			}()
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			if err != nil {
				log.Errorf("%s", err)
			}
		}

	}
	return engine.StatusOK
}
