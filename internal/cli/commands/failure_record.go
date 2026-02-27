package commands

import "github.com/zhuangbiaowei/LocalAIStack/internal/failure"

func recordFailureBestEffort(event failure.Event) {
	failure.RecordBestEffort(event)
}
