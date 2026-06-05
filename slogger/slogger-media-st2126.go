//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/
// This file is part of clog.

// package slogger_media adds new logging types to standardise the way in which
// long running media jobs (1hr transcodes, 5hr live shoots) are logges, split
// sharded and handled in the real world. It uses SMPTE ST 2126:2020 as a
// reference for the job status and error handling. It also uses RFC 7807 for
// error handling. The package is designed to be used in a micro-service
// environment where jobs are created, queued, run and completed in a
// distributed manner.

package slogger

import (
	"fmt"
	"html/template"
	"sync"
	"time"

	"github.com/longkai/rfc7807"
)

// this is the status that is exported externally
type JobStatusEnum string

// slight superset of SMPTE ST 2126
const (
	NEW       JobStatusEnum = "NEW"
	QUEUED    JobStatusEnum = "QUEUED"
	RUNNING   JobStatusEnum = "RUNNING"
	COMPLETED JobStatusEnum = "COMPLETED"
	FAILED    JobStatusEnum = "FAILED"
	CANCELLED JobStatusEnum = "CANCELLED"
)

// this is the internal state tracking micro-tasks
type StateEnum int

const (
	StateUnknown StateEnum = iota
	StateSeen    StateEnum = iota
	StateDeleted StateEnum = iota
	StateRunning StateEnum = iota
	StateDone    StateEnum = iota
)

type LogLevelCode int
type LogLevelName string
type LogLevelMap map[LogLevelName]LogLevelCode

const (
	FATAL          LogLevelCode = 100
	ERROR          LogLevelCode = 200
	WARN           LogLevelCode = 300
	INFO           LogLevelCode = 400
	DEBUG          LogLevelCode = 500
	FUNCTION_START LogLevelCode = 450
	FUNCTION_END   LogLevelCode = 450
	JOB_START      LogLevelCode = 400
	JOB_UPDATE     LogLevelCode = 400
	JOB_END        LogLevelCode = 400
)

var LogLevel = LogLevelMap{
	"FATAL":          FATAL,
	"ERROR":          ERROR,
	"WARN":           WARN,
	"INFO":           INFO,
	"DEBUG":          DEBUG,
	"FUNCTION_START": FUNCTION_START,
	"FUNCTION_END":   FUNCTION_END,
	"JOB_START":      JOB_START,
	"JOB_UPDATE":     JOB_UPDATE,
	"JOB_END":        JOB_END,
}

type URL string

type OutputInfo struct {
	LogLocation URL
}

type ErrorInfo *rfc7807.ProblemDetail

type JobInfo struct {
	//internal properties
	XjobId         int
	XfolderPath    URL
	XlockFilePath  URL
	XjobLogPath    URL
	XstudioLogPath URL
	Xmeta          string
	Xstate         StateEnum
	XfirstSeenAt   string
	XqueuedAt      string
	Xcli           string
	XDurationStr   string
	XAgeStr        string

	//the following parameters are external and follow SMPTE ST2126:2020
	Id              URL           // URL pointing to the job instance in the job processor
	Type            string        // Indicates the job type
	Profile         URL           //URL pointing to the job profile used by the job
	ProfileName     string        //Name of the job profile used by the job.
	Execution       URL           //URL pointing to the jobExecution instance in the job processor
	Assignment      URL           // URL pointing to the jobAssignment instance in the executing service
	Input           string        // Collection of input parameters that were provided in the job when it was created
	Status          JobStatusEnum //Status of the job
	Error           ErrorInfo     // Detailed info about the problem which caused the job. nil when not failed
	ActualStartDate string        // Date in ISO 8601 format when job was queued for processing
	ActualEndDate   string        //Date in ISO 8601 format when job completed, failed or canceled
	ActualDuration  int           //Job duration in milliseconds
	Output          OutputInfo    // Collection of output results of the job that was executed

}

type JobManagement struct {
	View          template.HTML
	JobRunning    *JobInfo
	Known         []JobInfo
	Queue         []*JobInfo
	Folder        string
	LockFileName  string
	LogCtlName    string
	LogStudioName string
	LogsFolder    string
	Xcli          string

	Wg sync.WaitGroup
}

// SMPTE ST2126 timestamp
func (j *JobInfo) TimeStamp() string {
	return time.Now().Format(time.DateTime)
}

// convert numerical id to full string "job0123"
func (j *JobInfo) IdString() string {
	return fmt.Sprintf("job%04d", j.XjobId)
}
