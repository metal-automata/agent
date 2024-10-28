package ctrl

import (
	"fmt"
	"time"

	"github.com/metal-automata/rivets/events"
)

const (
	subjectPrefix = "com.hollow.sh.controllers.commands"
)

func queueConfig(appName, facilityCode, natsURL, credsFile string) events.NatsOptions {
	// com.hollow.sh.controllers.commands.sandbox.servers.*
	consumerSubject := fmt.Sprintf(
		"%s.%s.servers.*",
		subjectPrefix,
		facilityCode,
	)

	return events.NatsOptions{
		URL:            natsURL,
		AppName:        appName,
		CredsFile:      credsFile,
		ConnectTimeout: time.Second * 60,
		Stream: &events.NatsStreamOptions{
			Name: "controllers",
			Subjects: []string{
				// com.hollow.sh.controllers.commands.>
				subjectPrefix + ".>",
			},
			Acknowledgements: true,
			DuplicateWindow:  time.Minute * 5,
			Retention:        "workQueue",
		},
		Consumer: &events.NatsConsumerOptions{
			Pull:              true,
			AckWait:           time.Minute * 5,
			MaxAckPending:     10,
			Name:              fmt.Sprintf("%s-%s", facilityCode, appName),
			QueueGroup:        appName,
			FilterSubject:     consumerSubject,
			SubscribeSubjects: []string{consumerSubject},
		},
		KVReplicationFactor: 3,
	}
}
