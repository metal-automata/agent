package ctrl

import (
	"fmt"
	"time"

	"github.com/metal-automata/rivets/condition"
	"github.com/metal-automata/rivets/events"
)

const (
	subjectPrefix = "com.hollow.sh.controllers.commands"
)

func queueConfig(appName, facilityCode, natsURL, credsFile string, conditionKinds []condition.Kind) events.NatsOptions {
	//	fmt.Println("appname:", appName)
	//	fmt.Println("consumer subjects: ")
	//consumerSubjects := []string{}
	//for _, kind := range conditionKinds {
	// prepare consumer subjects
	sub := fmt.Sprintf(
		// com.hollow.sh.controllers.commands.sandbox.servers.
		"%s.%s.servers.>",
		subjectPrefix,
		facilityCode,
	)

	fmt.Println(sub)
	///consumerSubjects = append(consumerSubjects, sub)
	//}

	fmt.Printf("filter subject: %s\n", subjectPrefix+".>")

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
			FilterSubject:     sub,
			SubscribeSubjects: []string{sub},
		},
		KVReplicationFactor: 3,
	}
}
