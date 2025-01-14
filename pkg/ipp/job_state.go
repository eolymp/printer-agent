package ipp

type JobState int

const (
	JobPending           JobState = 3
	JobPendingHeld       JobState = 4
	JobProcessing        JobState = 5
	JobProcessingStopped JobState = 6
	JobCanceled          JobState = 7
	JobAborted           JobState = 8
	JobCompleted         JobState = 9
)

func (s JobState) String() string {
	switch s {
	case JobPending:
		return "pending"
	case JobPendingHeld:
		return "pending-held"
	case JobProcessing:
		return "processing"
	case JobProcessingStopped:
		return "processing-stopped"
	case JobCanceled:
		return "canceled"
	case JobAborted:
		return "aborted"
	case JobCompleted:
		return "completed"
	default:
		return "unknown"
	}
}
